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

	// jumpClient is the bastion connection when this session was opened through
	// a ProxyJump; it is closed together with the session. nil otherwise.
	jumpClient *ssh.Client
	// tunnels owns all port-forward listeners and in-flight forwarded conns.
	tunnels *tunnelSet

	onData  DataHandler
	onClose CloseHandler

	// done is closed exactly once when the session terminates, signalling
	// background goroutines (e.g. keepalive) to exit.
	done chan struct{}

	mu     sync.Mutex
	closed bool
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	onData   DataHandler
	onClose  CloseHandler
	prompt   HostKeyPrompt
	onTunnel TunnelHandler
}

func NewManager(onData DataHandler, onClose CloseHandler, prompt HostKeyPrompt, onTunnel TunnelHandler) *Manager {
	return &Manager{
		sessions: map[string]*Session{},
		onData:   onData,
		onClose:  onClose,
		prompt:   prompt,
		onTunnel: onTunnel,
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

// OpenOptions carries everything needed to establish a session, including an
// optional resolved jump host (with its secrets) for ProxyJump. Port-forward
// definitions come from Host.Advanced.
type OpenOptions struct {
	SessionID    string
	Host         hosts.Host
	JumpHost     *hosts.Host // resolved bastion (with secrets), or nil
	Cols, Rows   int
	TimeoutSec   int
	KeepAliveSec int
}

// hostAddr returns the dial address "host:port" for h, defaulting port to 22.
func hostAddr(h hosts.Host) string {
	port := h.Port
	if port == 0 {
		port = 22
	}
	addr := h.Address
	if !strings.Contains(addr, ":") {
		addr = fmt.Sprintf("%s:%d", addr, port)
	}
	return addr
}

// clientConfig builds an ssh.ClientConfig for h using the shared host-key
// callback and handshake timeout.
func clientConfig(h hosts.Host, cb ssh.HostKeyCallback, timeout time.Duration) (*ssh.ClientConfig, error) {
	auth, err := buildAuth(h)
	if err != nil {
		return nil, err
	}
	return &ssh.ClientConfig{
		User:            h.User,
		Auth:            auth,
		HostKeyCallback: cb,
		Timeout:         timeout,
		ClientVersion:   "SSH-2.0-ssh-terminal",
	}, nil
}

// dial connects to h, optionally through a single jump host. On success it
// returns the target client and (when a bastion was used) the jump client,
// which the caller must close alongside the target.
func (m *Manager) dial(h hosts.Host, jump *hosts.Host, timeoutSec int) (*ssh.Client, *ssh.Client, error) {
	cb, err := m.hostKeyCallback()
	if err != nil {
		return nil, nil, err
	}
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	timeout := time.Duration(timeoutSec) * time.Second

	targetCfg, err := clientConfig(h, cb, timeout)
	if err != nil {
		return nil, nil, err
	}
	targetAddr := hostAddr(h)

	if jump == nil {
		client, derr := ssh.Dial("tcp", targetAddr, targetCfg)
		if derr != nil {
			return nil, nil, fmt.Errorf("dial: %w", derr)
		}
		return client, nil, nil
	}

	// ProxyJump: dial the bastion, then tunnel to the target through it.
	jumpCfg, err := clientConfig(*jump, cb, timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("proxy jump: bastion auth: %w", err)
	}
	jumpClient, err := ssh.Dial("tcp", hostAddr(*jump), jumpCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("proxy jump: dial bastion: %w", err)
	}
	conn, err := jumpClient.Dial("tcp", targetAddr)
	if err != nil {
		_ = jumpClient.Close()
		return nil, nil, fmt.Errorf("proxy jump: reach target via bastion: %w", err)
	}
	ncc, chans, reqs, err := ssh.NewClientConn(conn, targetAddr, targetCfg)
	if err != nil {
		_ = conn.Close()
		_ = jumpClient.Close()
		return nil, nil, fmt.Errorf("proxy jump: target handshake: %w", err)
	}
	return ssh.NewClient(ncc, chans, reqs), jumpClient, nil
}

// Open establishes a new SSH session and begins streaming.
// TimeoutSec is the TCP+SSH handshake timeout; 0 or negative falls back to 15 s.
// KeepAliveSec, when > 0, enables periodic keepalive@openssh.com requests at
// that interval; 0 or negative disables keepalive.
func (m *Manager) Open(opt OpenOptions) error {
	h := opt.Host
	cols, rows := opt.Cols, opt.Rows
	client, jumpClient, err := m.dial(h, opt.JumpHost, opt.TimeoutSec)
	if err != nil {
		return err
	}
	sess, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		if jumpClient != nil {
			_ = jumpClient.Close()
		}
		return fmt.Errorf("session: %w", err)
	}
	// closeClients tears down the target and (if any) bastion connection on an
	// early setup failure.
	closeClients := func() {
		_ = client.Close()
		if jumpClient != nil {
			_ = jumpClient.Close()
		}
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
		closeClients()
		return fmt.Errorf("pty: %w", err)
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		_ = sess.Close()
		closeClients()
		return err
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		_ = sess.Close()
		closeClients()
		return err
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		_ = sess.Close()
		closeClients()
		return err
	}
	if err := sess.Shell(); err != nil {
		_ = sess.Close()
		closeClients()
		return fmt.Errorf("shell: %w", err)
	}
	s := &Session{
		ID:         opt.SessionID,
		HostID:     h.ID,
		client:     client,
		session:    sess,
		stdin:      stdin,
		jumpClient: jumpClient,
		tunnels:    newTunnelSet(),
		onData:     m.onData,
		onClose:    m.onClose,
		done:       make(chan struct{}),
	}
	m.mu.Lock()
	m.sessions[opt.SessionID] = s
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
	if opt.KeepAliveSec > 0 {
		s.startKeepAlive(time.Duration(opt.KeepAliveSec) * time.Second)
	}
	// Start any enabled port-forward tunnels. Bind failures are reported via the
	// tunnel handler and never abort the session.
	s.startForwards(h.Advanced, m.onTunnel)
	return nil
}

// startKeepAlive periodically sends a keepalive@openssh.com global request on
// the client connection. It runs in its own goroutine and exits when the
// session's done channel is closed or a request fails (dead connection).
// It never touches stdin/stdout/stderr, so it cannot block the io pumps or
// the session-wait goroutine.
func (s *Session) startKeepAlive(interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-t.C:
				if _, _, err := s.client.SendRequest("keepalive@openssh.com", true, nil); err != nil {
					return
				}
			}
		}
	}()
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

// Run executes a one-off command on the session's SSH connection using a
// separate channel (via client.NewSession), NOT the interactive shell PTY, so
// it cannot inject into or disturb the terminal. It returns the command's
// combined stdout+stderr. A non-zero remote exit status is not treated as a
// hard failure: the collected output is still returned (a monitor command may
// exit non-zero if one sub-command fails while earlier output is valid). Only a
// missing session or a channel/transport error yields a non-nil error.
func (m *Manager) Run(sessionID, cmd string) ([]byte, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return nil, errors.New("session not found")
	}
	sess, err := s.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		// Non-zero exit still carries usable stdout; surface the bytes and let
		// the caller parse what it can. A transport/channel error returns as-is.
		var ee *ssh.ExitError
		if errors.As(err, &ee) {
			return out, nil
		}
		return out, err
	}
	return out, nil
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
	if s.done != nil {
		close(s.done)
	}
	s.mu.Unlock()
	// Tear down tunnels first (stops listeners + closes in-flight forwarded
	// connections) so every bound port is released, then the shell/client, then
	// the bastion connection.
	if s.tunnels != nil {
		s.tunnels.closeAll()
	}
	_ = s.stdin.Close()
	_ = s.session.Close()
	_ = s.client.Close()
	if s.jumpClient != nil {
		_ = s.jumpClient.Close()
	}
	if s.onClose != nil {
		s.onClose(s.ID, reason)
	}
}
