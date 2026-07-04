package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"

	"github.com/leungbzai-png/ssh-terminal/internal/bookmarks"
	"github.com/leungbzai-png/ssh-terminal/internal/config"
	"github.com/leungbzai-png/ssh-terminal/internal/hosts"
	"github.com/leungbzai-png/ssh-terminal/internal/keymgr"
	"github.com/leungbzai-png/ssh-terminal/internal/localfs"
	"github.com/leungbzai-png/ssh-terminal/internal/portable"
	"github.com/leungbzai-png/ssh-terminal/internal/redact"
	"github.com/leungbzai-png/ssh-terminal/internal/session"
	"github.com/leungbzai-png/ssh-terminal/internal/sftpx"
	"github.com/leungbzai-png/ssh-terminal/internal/sshconfig"
	"github.com/leungbzai-png/ssh-terminal/internal/sshsess"
)

// App is the bridge between Wails (Go) and the Vue frontend.
type App struct {
	ctx context.Context

	ssh  *sshsess.Manager
	sftp *sftpx.Manager

	pendingMu      sync.Mutex
	pendingPrompts map[string]chan bool

	// quitting becomes true once the user has confirmed closing the app,
	// allowing the next OnBeforeClose to fall through.
	quitting atomic.Bool
}

func NewApp() *App {
	return &App{
		sftp:           sftpx.NewManager(),
		pendingPrompts: map[string]chan bool{},
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	runtime.OnFileDrop(ctx, a.onFileDrop)
	a.ssh = sshsess.NewManager(
		func(id string, data []byte) {
			runtime.EventsEmit(a.ctx, "ssh:data:"+id, base64.StdEncoding.EncodeToString(data))
		},
		func(id string, reason string) {
			// Defence in depth: a close reason should never carry secret material,
			// but scrub any embedded PEM block before it reaches the frontend.
			runtime.EventsEmit(a.ctx, "ssh:close:"+id, redact.String(reason))
		},
		func(hostname, fp string) bool {
			ch := make(chan bool, 1)
			a.pendingMu.Lock()
			a.pendingPrompts[fp] = ch
			a.pendingMu.Unlock()
			runtime.EventsEmit(a.ctx, "ssh:hostkey", map[string]string{
				"hostname":    hostname,
				"fingerprint": fp,
			})
			ans := <-ch
			a.pendingMu.Lock()
			delete(a.pendingPrompts, fp)
			a.pendingMu.Unlock()
			return ans
		},
		// Tunnel status: forwards report bind success/failure here so the UI can
		// show a per-tunnel indicator. TunnelStatus carries no secret material.
		func(sessionID string, st sshsess.TunnelStatus) {
			runtime.EventsEmit(a.ctx, "ssh:tunnel:"+sessionID, st)
		},
	)
}

func (a *App) beforeClose(_ context.Context) bool {
	// Returning true means "block the close".
	if a.quitting.Load() {
		return false
	}
	s := config.Load()
	if !s.ConfirmCloseWithActiveSessions {
		return false
	}
	count := 0
	if a.ssh != nil {
		count = a.ssh.ActiveCount()
	}
	if count == 0 {
		return false
	}
	// Ask the frontend to show a confirmation modal.
	runtime.EventsEmit(a.ctx, "app:confirmClose", count)
	return true
}

func (a *App) shutdown(_ context.Context) {
	if a.ssh != nil {
		a.ssh.CloseAll()
	}
	if a.sftp != nil {
		a.sftp.CloseAll()
	}
}

// onFileDrop is registered with Wails and forwards drops to the frontend.
// The frontend decides which pane / session should receive them.
func (a *App) onFileDrop(x, y int, paths []string) {
	runtime.EventsEmit(a.ctx, "app:filedrop", map[string]any{
		"x":     x,
		"y":     y,
		"paths": paths,
	})
}

// --- Exposed API ---

func (a *App) AppInfo() map[string]string {
	return map[string]string{
		"name":    "SSH Terminal",
		"version": "1.0.0",
		"dataDir": portable.DataDir(),
		"baseDir": portable.BaseDir(),
	}
}

func (a *App) GetSettings() config.Settings         { return config.Load() }
func (a *App) SaveSettings(s config.Settings) error { return config.Save(s) }

func (a *App) ListHosts() ([]hosts.Host, error)            { return hosts.List() }
func (a *App) UpsertHost(h hosts.Host) (hosts.Host, error) { return hosts.Upsert(h) }
func (a *App) DeleteHost(id string) error                  { return hosts.Delete(id) }

// SshConfigPreviewEntry augments a parsed ssh_config entry with import-time
// metadata: whether its IdentityFile exists on disk and whether it duplicates
// an already-saved host. IdentityFile here is the ~-expanded absolute path.
type SshConfigPreviewEntry struct {
	sshconfig.Entry
	IdentityExists bool `json:"identityExists"`
	Duplicate      bool `json:"duplicate"`
}

// SshConfigImportResult summarizes an import run.
type SshConfigImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Names    []string `json:"names"`
}

// DefaultSshConfigPath returns the conventional ~/.ssh/config path (may be "").
func (a *App) DefaultSshConfigPath() string { return sshconfig.DefaultPath() }

// PickSshConfig lets the user browse for an ssh config file.
func (a *App) PickSshConfig() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select SSH config file"})
}

// PreviewSshConfig parses the ssh config at path (or the default location when
// path is empty) and returns entries annotated with existence/duplicate flags.
// It never modifies anything.
func (a *App) PreviewSshConfig(path string) ([]SshConfigPreviewEntry, error) {
	if path == "" {
		path = sshconfig.DefaultPath()
	}
	if path == "" {
		return nil, fmt.Errorf("无法确定 ~/.ssh/config 路径")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := sshconfig.Parse(f)
	if err != nil {
		return nil, err
	}
	existing, err := hosts.List()
	if err != nil {
		return nil, err
	}
	out := make([]SshConfigPreviewEntry, 0, len(entries))
	for _, e := range entries {
		if e.IdentityFile != "" {
			e.IdentityFile = sshconfig.ExpandUser(e.IdentityFile)
		}
		pe := SshConfigPreviewEntry{Entry: e}
		if e.IdentityFile != "" {
			if _, statErr := os.Stat(e.IdentityFile); statErr == nil {
				pe.IdentityExists = true
			} else {
				pe.Warnings = append(pe.Warnings, "密钥文件不存在: "+e.IdentityFile)
			}
		}
		pe.Duplicate = isDuplicateHost(existing, e)
		out = append(out, pe)
	}
	return out, nil
}

// findDuplicateHostID returns the ID of an existing host matching address+port+user
// (case-insensitive on address/user), or "", false when none matches. Port 0 is
// treated as 22 on both sides.
func findDuplicateHostID(existing []hosts.Host, address string, port int, user string) (string, bool) {
	if port == 0 {
		port = 22
	}
	for _, h := range existing {
		hp := h.Port
		if hp == 0 {
			hp = 22
		}
		if strings.EqualFold(h.Address, address) && hp == port &&
			strings.EqualFold(h.User, user) {
			return h.ID, true
		}
	}
	return "", false
}

// isDuplicateHost reports whether an ssh_config entry matches an existing host
// by address + port + user (case-insensitive on address/user).
func isDuplicateHost(existing []hosts.Host, e sshconfig.Entry) bool {
	_, ok := findDuplicateHostID(existing, e.HostName, e.Port, e.User)
	return ok
}

// ImportSshConfig saves the given entries as hosts. Entries whose HostName+Port+User
// already exist are skipped (never overwritten). IdentityFile is referenced by
// path only — no private key is copied into data/ and nothing is decrypted.
func (a *App) ImportSshConfig(entries []sshconfig.Entry) (SshConfigImportResult, error) {
	existing, err := hosts.List()
	if err != nil {
		return SshConfigImportResult{}, err
	}
	res := SshConfigImportResult{}
	for _, e := range entries {
		if e.HostName == "" {
			e.HostName = e.Alias
		}
		if isDuplicateHost(existing, e) {
			res.Skipped++
			continue
		}
		h := hosts.Host{
			Name:     e.Alias,
			Address:  e.HostName,
			Port:     e.Port,
			User:     e.User,
			AuthType: "password",
		}
		if e.IdentityFile != "" {
			// Reference the external key file directly; do NOT copy or decrypt it.
			h.AuthType = "key"
			h.KeyPath = sshconfig.ExpandUser(e.IdentityFile)
		}
		saved, uErr := hosts.Upsert(h)
		if uErr != nil {
			return res, uErr
		}
		// Track newly-saved host so later entries in the same batch dedupe against it.
		existing = append(existing, saved)
		res.Imported++
		res.Names = append(res.Names, saved.Name)
	}
	return res, nil
}

// --- Safe host export / import (v0.5.0) ---

// HostImportPreviewEntry augments an incoming SafeHost with import-time metadata:
// whether it duplicates an already-saved host (address+port+user) and, for
// key-auth hosts, whether the referenced external key path currently exists.
type HostImportPreviewEntry struct {
	hosts.SafeHost
	Duplicate bool `json:"duplicate"`
	KeyExists bool `json:"keyExists"`
}

// HostsImportPreview is returned by PreviewHostsImport: the picked file path plus
// the annotated hosts to be shown to the user before any change is made.
type HostsImportPreview struct {
	Path  string                   `json:"path"`
	Hosts []HostImportPreviewEntry `json:"hosts"`
}

// HostsImportResult summarizes an import run.
type HostsImportResult struct {
	Imported    int `json:"imported"`
	Skipped     int `json:"skipped"`
	Overwritten int `json:"overwritten"`
}

// ExportHosts writes a safe (no-secrets) host export to a user-chosen file and
// returns the saved path. The export contains only whitelisted, non-secret host
// metadata — never a password, passphrase, or private-key material. Returns ""
// (no error) if the user cancels the save dialog.
func (a *App) ExportHosts() (string, error) {
	data, err := hosts.MarshalExport()
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出主机（安全，不含密码或私钥）",
		DefaultFilename: "ssh-terminal-hosts.json",
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil // user cancelled
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// PreviewHostsImport lets the user pick a safe-export file and returns its parsed,
// annotated contents WITHOUT modifying anything. Returns an empty preview (path
// "") if the user cancels.
func (a *App) PreviewHostsImport() (HostsImportPreview, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "选择要导入的主机文件"})
	if err != nil {
		return HostsImportPreview{}, err
	}
	if path == "" {
		return HostsImportPreview{}, nil // user cancelled
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return HostsImportPreview{}, err
	}
	exp, err := hosts.ParseExport(data)
	if err != nil {
		return HostsImportPreview{}, err
	}
	existing, err := hosts.List()
	if err != nil {
		return HostsImportPreview{}, err
	}
	out := HostsImportPreview{Path: path, Hosts: make([]HostImportPreviewEntry, 0, len(exp.Hosts))}
	for _, sh := range exp.Hosts {
		_, dup := findDuplicateHostID(existing, sh.Address, sh.Port, sh.User)
		entry := HostImportPreviewEntry{SafeHost: sh, Duplicate: dup}
		if sh.AuthType == "key" && sh.KeyPath != "" {
			if _, statErr := os.Stat(sh.KeyPath); statErr == nil {
				entry.KeyExists = true
			}
		}
		out.Hosts = append(out.Hosts, entry)
	}
	return out, nil
}

// ImportHosts saves the given safe hosts. Duplicates (address+port+user) are
// skipped unless overwrite is true, in which case the existing record is updated
// in place. New hosts always receive a freshly minted ID. Imported hosts carry
// no secrets — passwords/passphrases must be added by the user afterwards. When
// overwriting, an existing encrypted password is preserved (never wiped).
func (a *App) ImportHosts(entries []hosts.SafeHost, overwrite bool) (HostsImportResult, error) {
	existing, err := hosts.List()
	if err != nil {
		return HostsImportResult{}, err
	}
	res := HostsImportResult{}
	for _, e := range entries {
		if e.Address == "" || e.User == "" {
			res.Skipped++
			continue
		}
		dupID, dup := findDuplicateHostID(existing, e.Address, e.Port, e.User)
		h := hosts.Host{
			Name:         e.Name,
			Address:      e.Address,
			Port:         e.Port,
			User:         e.User,
			AuthType:     e.AuthType,
			KeyPath:      e.KeyPath,
			ManagedKeyID: e.ManagedKeyID,
			Group:        e.Group,
			Note:         e.Note,
			Advanced:     e.Advanced,
		}
		if dup {
			if !overwrite {
				res.Skipped++
				continue
			}
			h.ID = dupID // update existing record in place (preserves stored secret)
			if _, uErr := hosts.Upsert(h); uErr != nil {
				return res, uErr
			}
			res.Overwritten++
			continue
		}
		saved, uErr := hosts.Upsert(h) // empty ID -> fresh ID
		if uErr != nil {
			return res, uErr
		}
		existing = append(existing, saved)
		res.Imported++
	}
	return res, nil
}

// --- Tab restore (v0.6.0) ---

// GetOpenTabs returns the saved-host tabs to restore on launch (non-secret:
// host id + display name only). Restored tabs are NOT auto-connected.
func (a *App) GetOpenTabs() []session.OpenTab { return session.Load() }

// SaveOpenTabs persists the current set of saved-host tabs. Quick Connect tabs
// (empty host id) are dropped and never written.
func (a *App) SaveOpenTabs(tabs []session.OpenTab) error { return session.Save(tabs) }

func (a *App) OpenSession(sessionID, hostID string, cols, rows int) error {
	h, err := hosts.Get(hostID)
	if err != nil {
		return err
	}
	jump, err := a.resolveJumpHost(h)
	if err != nil {
		return err
	}
	s := config.Load()
	err = a.ssh.Open(sshsess.OpenOptions{
		SessionID:    sessionID,
		Host:         h,
		JumpHost:     jump,
		Cols:         cols,
		Rows:         rows,
		TimeoutSec:   s.ConnectTimeoutSec,
		KeepAliveSec: keepAliveSecFrom(s),
	})
	if err != nil {
		return a.connectError(err, &h, jump)
	}
	return nil
}

// resolveJumpHost turns a host's ProxyJump config into a concrete jump host
// (with secrets, when it references a saved host). Manual mode is key-only by
// design — a manual bastion can never carry a password/passphrase. Returns
// (nil, nil) when the host has no ProxyJump configured.
func (a *App) resolveJumpHost(h hosts.Host) (*hosts.Host, error) {
	if h.Advanced == nil || h.Advanced.ProxyJump == nil {
		return nil, nil
	}
	pj := h.Advanced.ProxyJump
	switch pj.Mode {
	case hosts.ProxyJumpSavedHost:
		jh, err := hosts.Get(pj.JumpHostID) // decrypts the bastion's stored secrets
		if err != nil {
			return nil, fmt.Errorf("跳板机主机缺失或无法读取（proxy jump）")
		}
		return &jh, nil
	case hosts.ProxyJumpManual:
		if pj.KeyPath == "" {
			return nil, fmt.Errorf("手动跳板机需要密钥文件（proxy jump，不支持明文密码）")
		}
		jh := hosts.Host{
			Address:  pj.Address,
			Port:     pj.Port,
			User:     pj.User,
			AuthType: "key",
			KeyPath:  pj.KeyPath,
		}
		return &jh, nil
	default:
		return nil, fmt.Errorf("未知跳板机模式（proxy jump）: %q", pj.Mode)
	}
}

// connectError builds a user-facing connection error: it classifies the failure
// into a readable category and redacts any host/bastion secret that might have
// surfaced in the underlying error text. Secrets are scrubbed by VALUE using
// the actual credentials in scope here.
func (a *App) connectError(err error, h *hosts.Host, jump *hosts.Host) error {
	if err == nil {
		return nil
	}
	var secrets []string
	if h != nil {
		secrets = append(secrets, h.Password, h.Passphrase)
	}
	if jump != nil {
		secrets = append(secrets, jump.Password, jump.Passphrase)
	}
	msg := redact.String(err.Error(), secrets...)
	if diag := sshsess.DiagnoseError(err); diag != "" {
		return fmt.Errorf("%s：%s", diag, msg)
	}
	return fmt.Errorf("%s", msg)
}

// keepAliveSecFrom returns the effective keepalive interval in seconds for the
// given settings, or 0 when keepalive is disabled.
func keepAliveSecFrom(s config.Settings) int {
	if !s.KeepAliveEnabled {
		return 0
	}
	if s.KeepAliveIntervalSec <= 0 {
		return 30
	}
	return s.KeepAliveIntervalSec
}

// QuickConnectParams carries the ephemeral credentials for a Quick Connect
// session. These are NEVER persisted to hosts.json; the plaintext password /
// passphrase live only in memory for the lifetime of the request.
type QuickConnectParams struct {
	Address    string `json:"address"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	AuthType   string `json:"authType"` // "password" | "key" (external key file)
	Password   string `json:"password,omitempty"`
	KeyPath    string `json:"keyPath,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

// SshOpenQuick opens a one-off SSH session from ephemeral credentials without
// saving a host. It builds an in-memory hosts.Host (never written to disk) and
// reuses the normal session path, so keepalive and known_hosts verification
// apply identically. If the user wants to keep the host, the frontend calls
// UpsertHost separately (which encrypts secrets); this method must not persist.
func (a *App) SshOpenQuick(sessionID string, p QuickConnectParams, cols, rows int) error {
	if p.Address == "" || p.User == "" {
		return fmt.Errorf("address and user are required")
	}
	h := hosts.Host{
		Name:       p.Address,
		Address:    p.Address,
		Port:       p.Port,
		User:       p.User,
		AuthType:   p.AuthType,
		Password:   p.Password,
		KeyPath:    p.KeyPath,
		Passphrase: p.Passphrase,
	}
	s := config.Load()
	err := a.ssh.Open(sshsess.OpenOptions{
		SessionID:    sessionID,
		Host:         h,
		Cols:         cols,
		Rows:         rows,
		TimeoutSec:   s.ConnectTimeoutSec,
		KeepAliveSec: keepAliveSecFrom(s),
	})
	if err != nil {
		return a.connectError(err, &h, nil)
	}
	return nil
}

func (a *App) WriteSession(sessionID string, dataB64 string) error {
	data, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		return err
	}
	return a.ssh.Write(sessionID, data)
}

func (a *App) ResizeSession(sessionID string, cols, rows int) error {
	return a.ssh.Resize(sessionID, cols, rows)
}

func (a *App) CloseSession(sessionID string) error {
	a.sftp.Close(sessionID)
	return a.ssh.Close(sessionID)
}

func (a *App) ActiveSessionCount() int {
	if a.ssh == nil {
		return 0
	}
	return a.ssh.ActiveCount()
}

// ConfirmQuit is called by the frontend after the user confirms closing
// the app from the confirmation modal.
func (a *App) ConfirmQuit() {
	a.quitting.Store(true)
	go func() {
		// Tiny delay so the frontend has time to settle.
		time.Sleep(80 * time.Millisecond)
		runtime.Quit(a.ctx)
	}()
}

func (a *App) AnswerHostKey(fingerprint string, accept bool) {
	a.pendingMu.Lock()
	ch, ok := a.pendingPrompts[fingerprint]
	a.pendingMu.Unlock()
	if ok {
		select {
		case ch <- accept:
		default:
		}
	}
}

// SFTP
func (a *App) SftpList(sessionID, dir string) ([]sftpx.FileEntry, error) {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return nil, err
	}
	return a.sftp.List(sessionID, c, dir)
}
func (a *App) SftpCwd(sessionID string) (string, error) {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return "", err
	}
	return a.sftp.Cwd(sessionID, c)
}
func (a *App) SftpDownload(sessionID, remotePath, localPath string) error {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return err
	}
	return a.sftp.Download(sessionID, c, remotePath, localPath)
}
func (a *App) SftpUpload(sessionID, localPath, remotePath string) error {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return err
	}
	return a.sftp.Upload(sessionID, c, localPath, remotePath)
}

// SftpUploadPaths: batch upload, supports files and recursive directories.
// Emits "sftp:progress:<sessionID>" events during transfer with
// {transferred, total, currentFile}.
func (a *App) SftpUploadPaths(sessionID string, localPaths []string, remoteDir string) error {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return err
	}
	startedAt := time.Now()
	var lastEmit time.Time
	progress := func(transferred, total int64, current string) {
		now := time.Now()
		if now.Sub(lastEmit) < 80*time.Millisecond && transferred < total {
			return
		}
		lastEmit = now
		runtime.EventsEmit(a.ctx, "sftp:progress:"+sessionID, map[string]any{
			"transferred": transferred,
			"total":       total,
			"current":     current,
			"elapsedMs":   now.Sub(startedAt).Milliseconds(),
		})
	}
	err = a.sftp.UploadPaths(sessionID, c, localPaths, remoteDir, progress)
	runtime.EventsEmit(a.ctx, "sftp:done:"+sessionID, map[string]any{
		"ok":  err == nil,
		"err": errString(err),
	})
	return err
}

func errString(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

// xferEmitters builds progress/done callbacks for a tracked SFTP transfer on the
// dedicated "sftp:xfer:*" event namespace. This is intentionally separate from
// the drag-upload events ("sftp:progress"/"sftp:done") so the persistent
// SftpPanel listeners never collide with App.vue's transient drag listeners.
func (a *App) xferEmitters(sessionID, direction string) (sftpx.ProgressFn, func(error)) {
	startedAt := time.Now()
	var lastEmit time.Time
	progress := func(transferred, total int64, current string) {
		now := time.Now()
		if now.Sub(lastEmit) < 80*time.Millisecond && (total < 0 || transferred < total) {
			return
		}
		lastEmit = now
		runtime.EventsEmit(a.ctx, "sftp:xfer:progress:"+sessionID, map[string]any{
			"transferred": transferred,
			"total":       total,
			"current":     current,
			"direction":   direction,
			"elapsedMs":   now.Sub(startedAt).Milliseconds(),
		})
	}
	done := func(err error) {
		runtime.EventsEmit(a.ctx, "sftp:xfer:done:"+sessionID, map[string]any{
			"ok":        err == nil,
			"err":       errString(err),
			"direction": direction,
		})
	}
	return progress, done
}

// SftpDownloadTracked downloads a remote file to a local path, emitting progress
// on the sftp:xfer:* events for the SFTP panel to render.
func (a *App) SftpDownloadTracked(sessionID, remotePath, localPath string) error {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return err
	}
	progress, done := a.xferEmitters(sessionID, "download")
	err = a.sftp.DownloadWithProgress(sessionID, c, remotePath, localPath, progress)
	done(err)
	return err
}

// SftpUploadTracked uploads local files/dirs into remoteDir, emitting progress
// on the sftp:xfer:* events (used by the SFTP panel's upload button).
func (a *App) SftpUploadTracked(sessionID string, localPaths []string, remoteDir string) error {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return err
	}
	progress, done := a.xferEmitters(sessionID, "upload")
	err = a.sftp.UploadPaths(sessionID, c, localPaths, remoteDir, progress)
	done(err)
	return err
}

// SftpDownloadPathsTracked downloads remote files/dirs into localDir (recursive
// for directories), emitting progress on the same sftp:xfer:* events. Plural to
// mirror SftpUploadTracked and support multi-select in the two-pane UI.
func (a *App) SftpDownloadPathsTracked(sessionID string, remotePaths []string, localDir string) error {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return err
	}
	progress, done := a.xferEmitters(sessionID, "download")
	err = a.sftp.DownloadPaths(sessionID, c, remotePaths, localDir, progress)
	done(err)
	return err
}

// --- Remote path bookmarks (v0.7.0) ---

// ListBookmarks returns a host's remote-path bookmarks (non-secret).
func (a *App) ListBookmarks(hostID string) ([]bookmarks.Bookmark, error) {
	return bookmarks.List(hostID)
}

// AddBookmark saves a remote path for a host. name defaults to the path.
func (a *App) AddBookmark(hostID, name, remotePath string) (bookmarks.Bookmark, error) {
	return bookmarks.Add(hostID, name, remotePath)
}

// DeleteBookmark removes a bookmark by id.
func (a *App) DeleteBookmark(id string) error { return bookmarks.Delete(id) }

func (a *App) SftpDelete(sessionID, remotePath string) error {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return err
	}
	return a.sftp.Delete(sessionID, c, remotePath)
}
func (a *App) SftpDeleteRecursive(sessionID, remotePath string) error {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return err
	}
	return a.sftp.DeleteRecursive(sessionID, c, remotePath)
}
// TextPreview is the result of a read-only text preview.
type TextPreview struct {
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	TooLarge bool   `json:"tooLarge"`
	Binary   bool   `json:"binary"`
}

// previewMaxBytes caps how much of a remote file is fetched for preview.
const previewMaxBytes = 512 * 1024 // 512 KiB

// SftpPreviewText fetches a small remote file for read-only preview. Files over
// previewMaxBytes are reported as tooLarge (not read); non-UTF-8/binary files
// are reported as binary (not shown). Never modifies the remote file.
func (a *App) SftpPreviewText(sessionID, remotePath string) (TextPreview, error) {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return TextPreview{}, err
	}
	data, size, tooLarge, err := a.sftp.ReadFilePreview(sessionID, c, remotePath, previewMaxBytes)
	if err != nil {
		return TextPreview{}, err
	}
	if tooLarge {
		return TextPreview{Size: size, TooLarge: true}, nil
	}
	if !sftpx.IsProbablyText(data) {
		return TextPreview{Size: size, Binary: true}, nil
	}
	return TextPreview{Content: string(data), Size: size}, nil
}

func (a *App) SftpMkdir(sessionID, remotePath string) error {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return err
	}
	return a.sftp.Mkdir(sessionID, c, remotePath)
}
func (a *App) SftpRename(sessionID, oldPath, newPath string) error {
	c, err := a.ssh.Client(sessionID)
	if err != nil {
		return err
	}
	return a.sftp.Rename(sessionID, c, oldPath, newPath)
}

// --- Local filesystem browse (v1.1.0, SFTP two-pane local pane) ---
//
// These are read-only browse helpers for the local pane. They never write,
// never persist a path or listing, and touch no secret storage.

// LocalList returns the contents of a local directory (files, folders, symlinks).
func (a *App) LocalList(dir string) ([]localfs.Entry, error) {
	return localfs.List(dir)
}

// LocalHome returns the current user's home directory.
func (a *App) LocalHome() (string, error) {
	return localfs.Home()
}

// LocalRoots returns the local filesystem roots (Windows drive roots, or "/").
func (a *App) LocalRoots() ([]string, error) {
	return localfs.Roots()
}

// LocalParent returns the parent of dir and whether dir is a filesystem root
// (in which case the frontend should show the roots list instead).
func (a *App) LocalParent(dir string) (string, bool) {
	return localfs.Parent(dir)
}

func (a *App) PickFileToUpload() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select file to upload"})
}
func (a *App) PickFilesToUpload() ([]string, error) {
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select files to upload"})
}
func (a *App) PickSaveLocation(suggested string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save as",
		DefaultFilename: suggested,
	})
}
func (a *App) PickPrivateKey() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select private key"})
}

// -------- Keys (managed SSH keypairs) --------

func (a *App) ListKeys() ([]keymgr.Key, error) { return keymgr.List() }

func (a *App) GenerateKey(name, comment, keyType string, rsaBits int, passphrase string) (keymgr.Key, error) {
	return keymgr.Generate(name, comment, keymgr.KeyType(keyType), rsaBits, passphrase)
}

func (a *App) DeleteKey(id string) error { return keymgr.Delete(id) }

// ImportPrivateKey imports an existing private key file into the managed key
// store, encrypting it to data/keys/<id>.key.enc. The plaintext key is read on
// the Go side only (never crosses the bridge); the passphrase, if any, is used
// solely to validate a protected key and is never persisted.
func (a *App) ImportPrivateKey(name, comment, keyPath, passphrase string) (keymgr.Key, error) {
	return keymgr.ImportFromFile(name, comment, keyPath, passphrase)
}

func (a *App) GetPublicKey(id string) (string, error) { return keymgr.PublicKey(id) }

// DeployPublicKeyToHost opens a one-shot SSH connection to hostID using its
// stored credentials, and appends the managed key's public key to
// ~/.ssh/authorized_keys (idempotently).
func (a *App) DeployPublicKeyToHost(hostID, keyID string) error {
	h, err := hosts.Get(hostID)
	if err != nil {
		return err
	}
	auth, err := buildAuthForDeploy(h)
	if err != nil {
		return err
	}
	pub, err := keymgr.PublicKey(keyID)
	if err != nil {
		return err
	}
	pub = strings.TrimSpace(pub)
	if pub == "" {
		return fmt.Errorf("empty public key")
	}

	// Reuse known_hosts file from the manager so deploys verify identity.
	cb, err := a.ssh.HostKeyCallbackForDeploy()
	if err != nil {
		return err
	}

	port := h.Port
	if port == 0 {
		port = 22
	}
	addr := h.Address
	if !strings.Contains(addr, ":") {
		addr = fmt.Sprintf("%s:%d", addr, port)
	}
	s := config.Load()
	timeoutSec := s.ConnectTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	cfg := &ssh.ClientConfig{
		User:            h.User,
		Auth:            auth,
		HostKeyCallback: cb,
		Timeout:         time.Duration(timeoutSec) * time.Second,
		ClientVersion:   "SSH-2.0-ssh-terminal-deploy",
	}
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	// Escape any single quotes so we can wrap the key in single quotes safely.
	safe := strings.ReplaceAll(pub, "'", `'\''`)
	cmd := fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && (grep -qxF '%s' ~/.ssh/authorized_keys || echo '%s' >> ~/.ssh/authorized_keys)`,
		safe, safe,
	)
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("remote command failed: %v: %s", err, string(out))
	}
	return nil
}

// buildAuthForDeploy mirrors the auth logic in internal/sshsess.buildAuth.
// It lives here to avoid importing sshsess just for one-shot deploy operations.
// TODO: consolidate into a shared package if a third caller appears.
func buildAuthForDeploy(h hosts.Host) ([]ssh.AuthMethod, error) {
	switch h.AuthType {
	case "password":
		if h.Password == "" {
			return nil, fmt.Errorf("host has no stored password")
		}
		return []ssh.AuthMethod{ssh.Password(h.Password)}, nil
	case "key":
		data, err := os.ReadFile(h.KeyPath)
		if err != nil {
			return nil, err
		}
		var signer ssh.Signer
		if h.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(h.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(data)
		}
		if err != nil {
			return nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	case "managedKey":
		signer, err := keymgr.LoadSigner(h.ManagedKeyID, h.Passphrase)
		if err != nil {
			return nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	return nil, fmt.Errorf("unsupported auth type: %s", h.AuthType)
}
