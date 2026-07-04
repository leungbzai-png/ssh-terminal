//go:build integration
// +build integration

// Disposable in-process SSH server and localhost-only network helpers used by
// the build-tagged integration suite (integration_test.go).
//
// Everything here lives only behind the `integration` build tag, so a normal
// `go test ./...` never compiles or runs it. The server binds 127.0.0.1:0,
// generates a fresh ed25519 host key and a random password per instance, and is
// torn down via t.Cleanup. No credential is ever printed; callers redact.
package sshsess

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/leungbzai-png/ssh-terminal/internal/hosts"
)

// randHex returns n random bytes hex-encoded. Used for passwords and sentinels;
// the password value is never logged (see safeErr in integration_test.go).
func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---- in-process SSH server ----

type testServer struct {
	ln       net.Listener
	addr     string
	user     string
	password string
	signer   ssh.Signer

	connMu sync.Mutex
	conns  []*ssh.ServerConn
}

// newTestServer starts a disposable SSH server on 127.0.0.1:0 and registers its
// teardown with t.Cleanup, so listeners and accepted connections are always
// released when the (sub)test ends.
func newTestServer(t *testing.T) *testServer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &testServer{
		ln:       ln,
		addr:     ln.Addr().String(),
		user:     "qauser",
		password: randHex(16),
		signer:   signer,
	}
	go s.serve()
	t.Cleanup(s.close)
	return s
}

// close stops accepting and force-closes every accepted connection.
func (s *testServer) close() {
	_ = s.ln.Close()
	s.dropAll()
}

// dropAll forcibly closes every accepted SSH connection, simulating an
// unexpected network drop (not a user-initiated close).
func (s *testServer) dropAll() {
	s.connMu.Lock()
	conns := s.conns
	s.conns = nil
	s.connMu.Unlock()
	for _, c := range conns {
		_ = c.Close()
	}
}

func (s *testServer) serverConfig() *ssh.ServerConfig {
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == s.user && string(pass) == s.password {
				return nil, nil
			}
			return nil, fmt.Errorf("authentication failed")
		},
	}
	cfg.AddHostKey(s.signer)
	return cfg
}

func (s *testServer) serve() {
	for {
		nConn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(nConn)
	}
}

func (s *testServer) handleConn(nConn net.Conn) {
	sshConn, chans, reqs, err := ssh.NewServerConn(nConn, s.serverConfig())
	if err != nil {
		return
	}
	s.connMu.Lock()
	s.conns = append(s.conns, sshConn)
	s.connMu.Unlock()
	var myFwds []net.Listener
	go s.handleGlobalRequests(sshConn, reqs, &myFwds)
	for newCh := range chans {
		switch newCh.ChannelType() {
		case "session":
			go handleSession(newCh)
		case "direct-tcpip":
			go handleDirectTCPIP(newCh)
		default:
			_ = newCh.Reject(ssh.UnknownChannelType, "unsupported")
		}
	}
	// Connection closed: tear down any remote-forward listeners it created.
	for _, l := range myFwds {
		_ = l.Close()
	}
}

func handleSession(newCh ssh.NewChannel) {
	ch, reqs, err := newCh.Accept()
	if err != nil {
		return
	}
	go func() {
		for req := range reqs {
			switch req.Type {
			case "pty-req", "shell", "window-change", "env":
				if req.WantReply {
					_ = req.Reply(true, nil)
				}
			default:
				if req.WantReply {
					_ = req.Reply(false, nil)
				}
			}
		}
	}()
	_, _ = ch.Write([]byte("qa-shell ready\r\n"))
	go func() {
		_, _ = io.Copy(io.Discard, ch)
		_ = ch.Close()
	}()
}

type directTCPIP struct {
	DestAddr string
	DestPort uint32
	OrigAddr string
	OrigPort uint32
}

func handleDirectTCPIP(newCh ssh.NewChannel) {
	var d directTCPIP
	if err := ssh.Unmarshal(newCh.ExtraData(), &d); err != nil {
		_ = newCh.Reject(ssh.ConnectionFailed, "bad payload")
		return
	}
	ch, reqs, err := newCh.Accept()
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)
	remote, err := net.DialTimeout("tcp", net.JoinHostPort(d.DestAddr, strconv.Itoa(int(d.DestPort))), 5*time.Second)
	if err != nil {
		_ = ch.Close()
		return
	}
	pipeConn(ch, remote)
}

type tcpipForward struct {
	Addr string
	Port uint32
}

func (s *testServer) handleGlobalRequests(conn *ssh.ServerConn, reqs <-chan *ssh.Request, myFwds *[]net.Listener) {
	for req := range reqs {
		switch req.Type {
		case "tcpip-forward":
			var p tcpipForward
			if err := ssh.Unmarshal(req.Payload, &p); err != nil {
				_ = req.Reply(false, nil)
				continue
			}
			ln, err := net.Listen("tcp", net.JoinHostPort(p.Addr, strconv.Itoa(int(p.Port))))
			if err != nil {
				_ = req.Reply(false, nil) // server refuses the bind
				continue
			}
			bound := ln.Addr().(*net.TCPAddr).Port
			_ = req.Reply(true, ssh.Marshal(struct{ Port uint32 }{uint32(bound)}))
			*myFwds = append(*myFwds, ln)
			go s.acceptForward(conn, ln, p.Addr, uint32(bound))
		case "keepalive@openssh.com":
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
		default:
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
}

func (s *testServer) acceptForward(conn *ssh.ServerConn, ln net.Listener, bindAddr string, bindPort uint32) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			// Use the real originating port; x/crypto/ssh rejects origin port 0.
			origPort := uint32(0)
			if ta, ok := c.RemoteAddr().(*net.TCPAddr); ok {
				origPort = uint32(ta.Port)
			}
			payload := ssh.Marshal(directTCPIP{DestAddr: bindAddr, DestPort: bindPort, OrigAddr: "127.0.0.1", OrigPort: origPort})
			ch, reqs, err := conn.OpenChannel("forwarded-tcpip", payload)
			if err != nil {
				_ = c.Close()
				return
			}
			go ssh.DiscardRequests(reqs)
			pipeConn(ch, c)
		}(c)
	}
}

// pipeConn copies bidirectionally between two streams, closing both when either
// side ends. Named pipeConn to avoid colliding with Session.pipe in tunnel.go.
func pipeConn(a io.ReadWriteCloser, b io.ReadWriteCloser) {
	done := make(chan struct{}, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go cp(a, b)
	go cp(b, a)
	<-done
	_ = a.Close()
	_ = b.Close()
}

// hostFor builds a password-auth Host pointing at the test server.
func hostFor(s *testServer) hosts.Host {
	host, portStr, _ := net.SplitHostPort(s.addr)
	port, _ := strconv.Atoi(portStr)
	return hosts.Host{
		ID: "qa-" + randHex(4), Name: "qa-target", Address: host, Port: port,
		User: s.user, AuthType: "password", Password: s.password,
	}
}

// ---- localhost network helpers ----

// freePort reserves then releases an ephemeral port and returns its number.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return p
}

// portFree reports whether 127.0.0.1:port can currently be bound.
func portFree(port int) bool {
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

// waitPortFree waits up to d for the port to become bindable again (listener
// released). Readiness poll, not a fixed sleep.
func waitPortFree(port int, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if portFree(port) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// waitCond polls fn until it returns true or d elapses.
func waitCond(fn func() bool, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// httpGetVia dials addr (host:port) directly and does a minimal HTTP GET,
// returning whether the sentinel body was received.
func httpGetVia(addr, sentinel string) bool {
	c, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(4 * time.Second))
	_, _ = c.Write([]byte("GET / HTTP/1.0\r\nHost: x\r\n\r\n"))
	return readUntil(c, sentinel)
}

// readUntil accumulates from c until the sentinel appears or the stream ends.
func readUntil(c net.Conn, sentinel string) bool {
	var acc []byte
	buf := make([]byte, 4096)
	for {
		n, err := c.Read(buf)
		if n > 0 {
			acc = append(acc, buf[:n]...)
			if bytesContains(acc, sentinel) {
				return true
			}
		}
		if err != nil {
			return bytesContains(acc, sentinel)
		}
	}
}

func bytesContains(b []byte, sub string) bool {
	return len(sub) > 0 && indexOf(string(b), sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// socks5Get connects to the SOCKS5 proxy at proxyAddr, CONNECTs to target using
// the domain-name address form, and does a minimal HTTP GET, checking for the
// sentinel. Minimal client so we add no new module dependency.
func socks5Get(proxyAddr, targetHost string, targetPort int, sentinel string) (bool, error) {
	c, err := net.DialTimeout("tcp", proxyAddr, 3*time.Second)
	if err != nil {
		return false, err
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(4 * time.Second))
	// greeting: v5, 1 method, no-auth
	if _, err := c.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		return false, err
	}
	rep := make([]byte, 2)
	if _, err := io.ReadFull(c, rep); err != nil {
		return false, err
	}
	if rep[0] != 0x05 || rep[1] != 0x00 {
		return false, fmt.Errorf("socks greeting rejected: %v", rep)
	}
	// CONNECT with domain target (ATYP=0x03).
	req := []byte{0x05, 0x01, 0x00}
	host := []byte(targetHost)
	req = append(req, 0x03, byte(len(host)))
	req = append(req, host...)
	req = append(req, byte(targetPort>>8), byte(targetPort&0xff))
	if _, err := c.Write(req); err != nil {
		return false, err
	}
	resp := make([]byte, 10)
	if _, err := io.ReadFull(c, resp); err != nil {
		return false, err
	}
	if resp[1] != 0x00 {
		return false, fmt.Errorf("socks connect failed: rep=0x%02x", resp[1])
	}
	// tunnelled HTTP GET
	_, _ = c.Write([]byte("GET / HTTP/1.0\r\nHost: x\r\n\r\n"))
	return readUntil(c, sentinel), nil
}

// startHTTP starts a local HTTP server returning the sentinel on a free port and
// registers its shutdown with t.Cleanup.
func startHTTP(t *testing.T, sentinel string) int {
	t.Helper()
	port := freePort(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sentinel))
	})
	srv := &http.Server{Handler: mux}
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("start http service: %v", err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return port
}
