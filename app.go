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

	"github.com/leungbzai-png/ssh-terminal/internal/config"
	"github.com/leungbzai-png/ssh-terminal/internal/hosts"
	"github.com/leungbzai-png/ssh-terminal/internal/keymgr"
	"github.com/leungbzai-png/ssh-terminal/internal/portable"
	"github.com/leungbzai-png/ssh-terminal/internal/sftpx"
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
			runtime.EventsEmit(a.ctx, "ssh:close:"+id, reason)
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
		"version": "0.3.0",
		"dataDir": portable.DataDir(),
		"baseDir": portable.BaseDir(),
	}
}

func (a *App) GetSettings() config.Settings         { return config.Load() }
func (a *App) SaveSettings(s config.Settings) error { return config.Save(s) }

func (a *App) ListHosts() ([]hosts.Host, error)            { return hosts.List() }
func (a *App) UpsertHost(h hosts.Host) (hosts.Host, error) { return hosts.Upsert(h) }
func (a *App) DeleteHost(id string) error                  { return hosts.Delete(id) }

func (a *App) OpenSession(sessionID, hostID string, cols, rows int) error {
	h, err := hosts.Get(hostID)
	if err != nil {
		return err
	}
	s := config.Load()
	return a.ssh.Open(sessionID, h, cols, rows, s.ConnectTimeoutSec, keepAliveSecFrom(s))
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
	return a.ssh.Open(sessionID, h, cols, rows, s.ConnectTimeoutSec, keepAliveSecFrom(s))
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
