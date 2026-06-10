// Package sshsess wraps a single interactive SSH shell session with PTY,
// strict known_hosts verification, and io streaming to a frontend channel.
package sshsess

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/leungbzai-png/ssh-terminal/internal/hosts"
	"github.com/leungbzai-png/ssh-terminal/internal/keymgr"
	"github.com/leungbzai-png/ssh-terminal/internal/portable"
)

type DataHandler func(id string, data []byte)
type CloseHandler func(id string, reason string)

// HostKeyPrompt is called when the server's host key is unknown.
// Return true to accept (and pin) the key.
type HostKeyPrompt func(hostname string, fingerprint string) bool

type Session struct {
	ID      string
	HostID  string
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser

	onData  DataHandler
	onClose CloseHandler

	mu     sync.Mutex
	closed bool
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	onData   DataHandler
	onClose  CloseHandler
	prompt   HostKeyPrompt
}

func NewManager(onData DataHandler, onClose CloseHandler, prompt HostKeyPrompt) *Manager {
	return &Manager{
		sessions: map[string]*Session{},
		onData:   onData,
		onClose:  onClose,
		prompt:   prompt,
	}
}

func knownHostsPath() string { return portable.DataPath("known_hosts") }

// ensureKnownHostsFile makes sure the file exists so knownhosts.New doesn't error.
func ensureKnownHostsFile() error {
	p := knownHostsPath()
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(p, []byte{}, 0o600)
	}
	return nil
}

// HostKeyCallbackForDeploy exposes the same known_hosts callback used for sessions,
// so one-shot operations (e.g. key deployment) get identical verification.
func (m *Manager) HostKeyCallbackForDeploy() (ssh.HostKeyCallback, error) {
	return m.hostKeyCallback()
}

// hostKeyCallback builds a callback that consults known_hosts and prompts
// the user (via the configured HostKeyPrompt) on first-seen keys.
func (m *Manager) hostKeyCallback() (ssh.HostKeyCallback, error) {
	if err := ensureKnownHostsFile(); err != nil {
		return nil, err
	}
	verify, err := knownhosts.New(knownHostsPath())
	if err != nil {
		return nil, err
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := verify(hostname, remote, key); err == nil {
			return nil
		} else if kerr, ok := err.(*knownhosts.KeyError); ok && len(kerr.Want) > 0 {
			// Key MISMATCH for known host - hard fail.
			return fmt.Errorf("host key mismatch for %s (possible MITM)", hostname)
		}
		// Unknown host: prompt user.
		fp := ssh.FingerprintSHA256(key)
		if m.prompt != nil && m.prompt(hostname, fp) {
			return appendKnownHost(hostname, remote, key)
		}
		return errors.New("host key not trusted by user")
	}, nil
}

func appendKnownHost(hostname string, remote net.Addr, key ssh.PublicKey) error {
	line := knownhosts.Line([]string{knownhosts.Normalize(hostname), knownhosts.Normalize(remote.String())}, key)
	f, err := os.OpenFile(knownHostsPath(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	return err
}

func buildAuth(h hosts.Host) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	switch h.AuthType {
	case "password":
		if h.Password == "" {
			return nil, errors.New("password authentication selected but no password set")
		}
		methods = append(methods, ssh.Password(h.Password))
	case "key":
		if h.KeyPath == "" {
			return nil, errors.New("key authentication selected but no key path set")
		}
		data, err := os.ReadFile(h.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("read key: %w", err)
		}
		var signer ssh.Signer
		if h.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(h.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(data)
		}
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	case "managedKey":
		if h.ManagedKeyID == "" {
			return nil, errors.New("managed key authentication selected but no key chosen")
		}
		signer, err := keymgr.LoadSigner(h.ManagedKeyID, h.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("load managed key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", h.AuthType)
	}
	return methods, nil
}

// Open establishes a new SSH session and begins streaming.
func (m *Manager) Open(sessionID string, h hosts.Host, cols, rows int) error {
	auth, err := buildAuth(h)
	if err != nil {
		return err
	}
	cb, err := m.hostKeyCallback()
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
	cfg := &ssh.ClientConfig{
		User:            h.User,
		Auth:            auth,
		HostKeyCallback: cb,
		Timeout:         15 * time.Second,
		ClientVersion:   "SSH-2.0-ssh-terminal",
	}
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	sess, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("session: %w", err)
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if cols <= 0 {
		cols = 120
	}
	if rows <= 0 {
		rows = 30
	}
	if err := sess.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		_ = sess.Close()
		_ = client.Close()
		return fmt.Errorf("pty: %w", err)
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		_ = sess.Close()
		_ = client.Close()
		return err
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		_ = sess.Close()
		_ = client.Close()
		return err
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		_ = sess.Close()
		_ = client.Close()
		return err
	}
	if err := sess.Shell(); err != nil {
		_ = sess.Close()
		_ = client.Close()
		return fmt.Errorf("shell: %w", err)
	}
	s := &Session{
		ID:      sessionID,
		HostID:  h.ID,
		client:  client,
		session: sess,
		stdin:   stdin,
		onData:  m.onData,
		onClose: m.onClose,
	}
	m.mu.Lock()
	m.sessions[sessionID] = s
	m.mu.Unlock()

	go s.pump(stdout)
	go s.pump(stderr)
	go func() {
		err := sess.Wait()
		reason := ""
		if err != nil {
			reason = err.Error()
		}
		s.closeWithReason(reason)
	}()
	return nil
}

func (s *Session) pump(r io.Reader) {
	buf := make([]byte, 8192)
	for {
		n, err := r.Read(buf)
		if n > 0 && s.onData != nil {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			s.onData(s.ID, chunk)
		}
		if err != nil {
			return
		}
	}
}

// Write sends user keystrokes to the remote shell.
func (m *Manager) Write(sessionID string, data []byte) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return errors.New("session not found")
	}
	_, err := s.stdin.Write(data)
	return err
}

// Resize forwards terminal size changes to the remote PTY.
func (m *Manager) Resize(sessionID string, cols, rows int) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return errors.New("session not found")
	}
	return s.session.WindowChange(rows, cols)
}

// Close terminates a session.
func (m *Manager) Close(sessionID string) error {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}
	s.closeWithReason("user closed")
	return nil
}

// ActiveCount returns the number of currently open sessions.
func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// CloseAll terminates every active session (used on app shutdown).
func (m *Manager) CloseAll() {
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = map[string]*Session{}
	m.mu.Unlock()
	for _, s := range sessions {
		s.closeWithReason("app shutdown")
	}
}

// Client exposes the underlying ssh.Client (for sftp).
func (m *Manager) Client(sessionID string) (*ssh.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	return s.client, nil
}

func (s *Session) closeWithReason(reason string) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	_ = s.stdin.Close()
	_ = s.session.Close()
	_ = s.client.Close()
	if s.onClose != nil {
		s.onClose(s.ID, reason)
	}
}
