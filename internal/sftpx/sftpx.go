// Package sftpx provides file operations over an existing SSH session.
package sftpx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime int64  `json:"modTime"`
	IsDir   bool   `json:"isDir"`
	IsLink  bool   `json:"isLink"`
}

// ProgressFn is called periodically during transfers.
// transferred: bytes done across all files in the current batch.
// total: total bytes if known, else -1.
// currentFile: human-readable current item.
type ProgressFn func(transferred, total int64, currentFile string)

type Manager struct {
	mu      sync.Mutex
	clients map[string]*sftp.Client // keyed by sessionID
}

func NewManager() *Manager { return &Manager{clients: map[string]*sftp.Client{}} }

func (m *Manager) client(sessionID string, sshClient *ssh.Client) (*sftp.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clients[sessionID]; ok {
		return c, nil
	}
	c, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, err
	}
	m.clients[sessionID] = c
	return c, nil
}

func (m *Manager) Close(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clients[sessionID]; ok {
		_ = c.Close()
		delete(m.clients, sessionID)
	}
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.clients {
		_ = c.Close()
		delete(m.clients, id)
	}
}

// List directory contents.
func (m *Manager) List(sessionID string, sshClient *ssh.Client, dir string) ([]FileEntry, error) {
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return nil, err
	}
	if dir == "" {
		dir, err = c.Getwd()
		if err != nil {
			return nil, err
		}
	}
	infos, err := c.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]FileEntry, 0, len(infos))
	for _, fi := range infos {
		e := FileEntry{
			Name:    fi.Name(),
			Path:    path.Join(dir, fi.Name()),
			Size:    fi.Size(),
			Mode:    fi.Mode().String(),
			ModTime: fi.ModTime().Unix(),
			IsDir:   fi.IsDir(),
			IsLink:  fi.Mode()&os.ModeSymlink != 0,
		}
		out = append(out, e)
	}
	return out, nil
}

// Cwd returns the user's home (initial) directory.
func (m *Manager) Cwd(sessionID string, sshClient *ssh.Client) (string, error) {
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return "", err
	}
	return c.Getwd()
}

// Download a remote file to a local path.
func (m *Manager) Download(sessionID string, sshClient *ssh.Client, remotePath, localPath string) error {
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return err
	}
	src, err := c.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

// DownloadWithProgress downloads a remote file to a local path, reporting byte
// progress via the callback. total is the remote file size, or -1 if unknown.
func (m *Manager) DownloadWithProgress(sessionID string, sshClient *ssh.Client, remotePath, localPath string, progress ProgressFn) error {
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return err
	}
	src, err := c.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()
	var total int64 = -1
	if fi, serr := c.Stat(remotePath); serr == nil {
		total = fi.Size()
	}
	dst, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	buf := make([]byte, 256*1024) // 256 KiB
	var transferred int64
	name := path.Base(remotePath)
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			transferred += int64(n)
			if progress != nil {
				progress(transferred, total, name)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	if progress != nil {
		progress(transferred, total, "")
	}
	return nil
}

// Upload a local file to a remote path.
func (m *Manager) Upload(sessionID string, sshClient *ssh.Client, localPath, remotePath string) error {
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return err
	}
	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := c.Create(remotePath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

// UploadPaths uploads a batch of local files and/or directories into remoteDir.
// Directories are walked recursively. Progress (if non-nil) is invoked every
// ~256 KiB of transferred data.
func (m *Manager) UploadPaths(
	sessionID string, sshClient *ssh.Client,
	localPaths []string, remoteDir string,
	progress ProgressFn,
) error {
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return err
	}

	// Phase 1: compute total bytes and collect file plans.
	type plan struct {
		localPath  string
		remotePath string
		size       int64
		isDir      bool
	}
	var plans []plan
	var total int64
	for _, lp := range localPaths {
		fi, err := os.Stat(lp)
		if err != nil {
			return err
		}
		base := filepath.Base(lp)
		rootRemote := joinPosix(remoteDir, base)
		if !fi.IsDir() {
			plans = append(plans, plan{lp, rootRemote, fi.Size(), false})
			total += fi.Size()
			continue
		}
		// Walk the directory.
		plans = append(plans, plan{lp, rootRemote, 0, true})
		err = filepath.Walk(lp, func(p string, info os.FileInfo, werr error) error {
			if werr != nil {
				return werr
			}
			if p == lp {
				return nil
			}
			rel, _ := filepath.Rel(lp, p)
			rel = filepath.ToSlash(rel)
			rp := joinPosix(rootRemote, rel)
			if info.IsDir() {
				plans = append(plans, plan{p, rp, 0, true})
			} else {
				plans = append(plans, plan{p, rp, info.Size(), false})
				total += info.Size()
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	var transferred int64
	for _, pl := range plans {
		if pl.isDir {
			if err := mkdirAll(c, pl.remotePath); err != nil {
				return err
			}
			if progress != nil {
				progress(transferred, total, "mkdir "+pl.remotePath)
			}
			continue
		}
		if err := mkdirAll(c, path.Dir(pl.remotePath)); err != nil {
			return err
		}
		if err := copyFileWithProgress(c, pl.localPath, pl.remotePath, &transferred, total, progress); err != nil {
			return err
		}
	}
	if progress != nil {
		progress(transferred, total, "")
	}
	return nil
}

func copyFileWithProgress(c *sftp.Client, localPath, remotePath string,
	transferred *int64, total int64, progress ProgressFn) error {
	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := c.Create(remotePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	buf := make([]byte, 256*1024) // 256 KiB
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			*transferred += int64(n)
			if progress != nil {
				progress(*transferred, total, filepath.Base(localPath))
			}
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}

// DownloadPaths downloads a batch of remote files and/or directories into
// localDir. A remote file lands at localDir/<basename>; a remote directory is
// recreated recursively under localDir/<dirname>/... Remote operations use
// POSIX paths (via the sftp client); local materialization uses filepath with
// filepath.FromSlash, and every target is confined to localDir (path-traversal
// safe). Progress (if non-nil) is invoked every ~256 KiB, mirroring UploadPaths.
func (m *Manager) DownloadPaths(
	sessionID string, sshClient *ssh.Client,
	remotePaths []string, localDir string,
	progress ProgressFn,
) error {
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return err
	}

	// Phase 1: plan (compute total bytes + local targets, walking remote dirs).
	type plan struct {
		remotePath string
		localPath  string
		size       int64
		isDir      bool
	}
	var plans []plan
	var total int64
	for _, rp := range remotePaths {
		fi, err := c.Stat(rp)
		if err != nil {
			return err
		}
		base := path.Base(rp)
		localRoot, err := safeLocalJoin(localDir, base)
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			plans = append(plans, plan{rp, localRoot, fi.Size(), false})
			total += fi.Size()
			continue
		}
		plans = append(plans, plan{rp, localRoot, 0, true})
		walker := c.Walk(rp)
		for walker.Step() {
			if werr := walker.Err(); werr != nil {
				return werr
			}
			wp := walker.Path()
			if wp == rp {
				continue
			}
			rel := strings.TrimPrefix(strings.TrimPrefix(wp, rp), "/")
			lp, jerr := safeLocalJoin(localRoot, rel)
			if jerr != nil {
				return jerr
			}
			info := walker.Stat()
			if info.IsDir() {
				plans = append(plans, plan{wp, lp, 0, true})
			} else {
				plans = append(plans, plan{wp, lp, info.Size(), false})
				total += info.Size()
			}
		}
	}

	var transferred int64
	for _, pl := range plans {
		if pl.isDir {
			if err := os.MkdirAll(pl.localPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(pl.localPath), 0o755); err != nil {
			return err
		}
		if err := downloadFileWithProgress(c, pl.remotePath, pl.localPath, &transferred, total, progress); err != nil {
			return err
		}
	}
	if progress != nil {
		progress(transferred, total, "")
	}
	return nil
}

// safeLocalJoin joins a POSIX-style relative path under base, converting it to
// the local separator, and refuses any result that escapes base (defends
// against ".." in server-reported names).
func safeLocalJoin(base, rel string) (string, error) {
	local := filepath.Join(base, filepath.FromSlash(rel))
	within, err := filepath.Rel(base, local)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing unsafe download path: %q", rel)
	}
	return local, nil
}

func downloadFileWithProgress(c *sftp.Client, remotePath, localPath string,
	transferred *int64, total int64, progress ProgressFn) error {
	src, err := c.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	buf := make([]byte, 256*1024) // 256 KiB
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			*transferred += int64(n)
			if progress != nil {
				progress(*transferred, total, path.Base(remotePath))
			}
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}

func mkdirAll(c *sftp.Client, p string) error {
	if p == "" || p == "/" || p == "." {
		return nil
	}
	if fi, err := c.Stat(p); err == nil {
		if fi.IsDir() {
			return nil
		}
		return errors.New("path exists and is not a directory: " + p)
	}
	parent := path.Dir(p)
	if parent != p {
		if err := mkdirAll(c, parent); err != nil {
			return err
		}
	}
	return c.Mkdir(p)
}

func joinPosix(parts ...string) string {
	out := ""
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if out == "" {
			out = p
			continue
		}
		if !strings.HasSuffix(out, "/") {
			out += "/"
		}
		out += strings.TrimPrefix(p, "/")
	}
	return out
}

// Delete removes a remote file or empty directory.
func (m *Manager) Delete(sessionID string, sshClient *ssh.Client, remotePath string) error {
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return err
	}
	fi, err := c.Stat(remotePath)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return c.RemoveDirectory(remotePath)
	}
	return c.Remove(remotePath)
}

// DeleteRecursive removes a remote file or directory tree recursively.
// Rejected paths: empty, "/", or "." (safety guard).
func (m *Manager) DeleteRecursive(sessionID string, sshClient *ssh.Client, remotePath string) error {
	if remotePath == "" || remotePath == "/" || remotePath == "." {
		return errors.New("refusing to delete root or empty path")
	}
	// Normalize: strip trailing slash.
	for len(remotePath) > 1 && strings.HasSuffix(remotePath, "/") {
		remotePath = remotePath[:len(remotePath)-1]
	}
	if remotePath == "/" {
		return errors.New("refusing to delete root")
	}
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return err
	}
	fi, err := c.Stat(remotePath)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return c.Remove(remotePath)
	}
	return c.RemoveAll(remotePath)
}

// Mkdir creates a remote directory.
func (m *Manager) Mkdir(sessionID string, sshClient *ssh.Client, remotePath string) error {
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return err
	}
	return c.Mkdir(remotePath)
}

// Rename moves a remote file.
func (m *Manager) Rename(sessionID string, sshClient *ssh.Client, oldPath, newPath string) error {
	c, err := m.client(sessionID, sshClient)
	if err != nil {
		return err
	}
	return c.Rename(oldPath, newPath)
}

var ErrNoSession = errors.New("no sftp session")

// IsProbablyText reports whether data looks like decodable UTF-8 text (safe to
// show in a read-only preview). A NUL byte or invalid UTF-8 is treated as
// binary. Empty input is considered text.
func IsProbablyText(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	return utf8.Valid(data)
}

// ReadFilePreview reads a remote file for read-only preview. If the file is
// larger than maxBytes it is NOT read and tooLarge is true (the caller should
// suggest downloading instead). Directories are rejected.
func (m *Manager) ReadFilePreview(sessionID string, sshClient *ssh.Client, remotePath string, maxBytes int64) (data []byte, size int64, tooLarge bool, err error) {
	c, cerr := m.client(sessionID, sshClient)
	if cerr != nil {
		return nil, 0, false, cerr
	}
	fi, serr := c.Stat(remotePath)
	if serr != nil {
		return nil, 0, false, serr
	}
	if fi.IsDir() {
		return nil, 0, false, errors.New("cannot preview a directory")
	}
	size = fi.Size()
	if size > maxBytes {
		return nil, size, true, nil
	}
	f, oerr := c.Open(remotePath)
	if oerr != nil {
		return nil, size, false, oerr
	}
	defer f.Close()
	data, err = io.ReadAll(io.LimitReader(f, maxBytes))
	if err != nil {
		return nil, size, false, err
	}
	return data, size, false, nil
}
