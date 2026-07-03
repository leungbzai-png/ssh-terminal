package sshsess

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/leungbzai-png/ssh-terminal/internal/hosts"
)

// Port forwarding tunnels (v0.8.0).
//
// A tunnelSet owns every listener and in-flight forwarded connection created
// for a session. closeAll() stops accepting and tears down live connections, so
// closing a tab / disconnecting releases all bound ports and leaks no
// goroutines (accept loops exit when their listener closes).

// TunnelStatus reports the outcome of starting one tunnel, so the UI can show a
// bound/failed indicator. It never carries secret material.
type TunnelStatus struct {
	Kind   string `json:"kind"` // "local" | "remote" | "dynamic"
	Name   string `json:"name"`
	Listen string `json:"listen"` // bind address, e.g. "127.0.0.1:8080"
	OK     bool   `json:"ok"`
	Err    string `json:"err,omitempty"`
}

// TunnelHandler receives tunnel status events for a session.
type TunnelHandler func(sessionID string, st TunnelStatus)

type tunnelSet struct {
	mu        sync.Mutex
	listeners []net.Listener
	conns     map[net.Conn]struct{}
	closed    bool
}

func newTunnelSet() *tunnelSet {
	return &tunnelSet{conns: map[net.Conn]struct{}{}}
}

// addListener registers a listener. Returns false (without registering) if the
// set is already closed, so a late listener is not leaked.
func (ts *tunnelSet) addListener(l net.Listener) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.closed {
		return false
	}
	ts.listeners = append(ts.listeners, l)
	return true
}

// track registers an in-flight connection. Returns false if the set is closed.
func (ts *tunnelSet) track(c net.Conn) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.closed {
		return false
	}
	ts.conns[c] = struct{}{}
	return true
}

func (ts *tunnelSet) untrack(c net.Conn) {
	ts.mu.Lock()
	delete(ts.conns, c)
	ts.mu.Unlock()
}

// closeAll stops every listener and closes every live connection. Idempotent.
func (ts *tunnelSet) closeAll() {
	ts.mu.Lock()
	if ts.closed {
		ts.mu.Unlock()
		return
	}
	ts.closed = true
	listeners := ts.listeners
	ts.listeners = nil
	conns := make([]net.Conn, 0, len(ts.conns))
	for c := range ts.conns {
		conns = append(conns, c)
	}
	ts.conns = map[net.Conn]struct{}{}
	ts.mu.Unlock()

	for _, l := range listeners {
		_ = l.Close()
	}
	for _, c := range conns {
		_ = c.Close()
	}
}

// localBindOr returns the bind host, defaulting to 127.0.0.1 when empty so a
// tunnel is never accidentally exposed on all interfaces.
func localBindOr(h string) string {
	if h == "" {
		return "127.0.0.1"
	}
	return h
}

// startForwards launches every enabled forward in adv. Bind failures are
// reported via onTunnel and never abort the session or other tunnels.
func (s *Session) startForwards(adv *hosts.AdvancedSSH, onTunnel TunnelHandler) {
	if adv == nil {
		return
	}
	for _, f := range adv.LocalForwards {
		if f.Enabled {
			s.startLocalForward(f, onTunnel)
		}
	}
	for _, f := range adv.DynamicForwards {
		if f.Enabled {
			s.startDynamicForward(f, onTunnel)
		}
	}
	for _, f := range adv.RemoteForwards {
		if f.Enabled {
			s.startRemoteForward(f, onTunnel)
		}
	}
}

func (s *Session) emitTunnel(onTunnel TunnelHandler, st TunnelStatus) {
	if onTunnel != nil {
		onTunnel(s.ID, st)
	}
}

func (s *Session) startLocalForward(f hosts.Forward, onTunnel TunnelHandler) {
	bind := net.JoinHostPort(localBindOr(f.LocalHost), strconv.Itoa(f.LocalPort))
	l, err := net.Listen("tcp", bind)
	if err != nil {
		s.emitTunnel(onTunnel, TunnelStatus{Kind: "local", Name: f.Name, Listen: bind, OK: false,
			Err: fmt.Sprintf("local forward: listen %s: %v", bind, err)})
		return
	}
	if !s.tunnels.addListener(l) {
		_ = l.Close()
		return
	}
	s.emitTunnel(onTunnel, TunnelStatus{Kind: "local", Name: f.Name, Listen: bind, OK: true})
	target := net.JoinHostPort(f.RemoteHost, strconv.Itoa(f.RemotePort))
	go s.acceptLoop(l, func(local net.Conn) {
		remote, derr := s.client.Dial("tcp", target)
		if derr != nil {
			_ = local.Close()
			return
		}
		s.pipe(local, remote)
	})
}

func (s *Session) startDynamicForward(f hosts.DynamicForward, onTunnel TunnelHandler) {
	bind := net.JoinHostPort(localBindOr(f.LocalHost), strconv.Itoa(f.LocalPort))
	l, err := net.Listen("tcp", bind)
	if err != nil {
		s.emitTunnel(onTunnel, TunnelStatus{Kind: "dynamic", Name: f.Name, Listen: bind, OK: false,
			Err: fmt.Sprintf("dynamic forward: listen %s: %v", bind, err)})
		return
	}
	if !s.tunnels.addListener(l) {
		_ = l.Close()
		return
	}
	s.emitTunnel(onTunnel, TunnelStatus{Kind: "dynamic", Name: f.Name, Listen: bind, OK: true})
	go s.acceptLoop(l, s.serveSocks)
}

// serveSocks handles a single SOCKS5 client connection, dialling the requested
// target through the SSH client.
func (s *Session) serveSocks(local net.Conn) {
	if err := socksNegotiate(local); err != nil {
		_ = local.Close()
		return
	}
	target, err := socksReadRequest(local)
	if err != nil {
		_ = socksWriteReply(local, socksRepErr)
		_ = local.Close()
		return
	}
	remote, err := s.client.Dial("tcp", target)
	if err != nil {
		_ = socksWriteReply(local, socksRepErr)
		_ = local.Close()
		return
	}
	if err := socksWriteReply(local, socksRepOK); err != nil {
		_ = local.Close()
		_ = remote.Close()
		return
	}
	s.pipe(local, remote)
}

func (s *Session) startRemoteForward(f hosts.Forward, onTunnel TunnelHandler) {
	// Default the remote bind to 127.0.0.1 for safety; the server's GatewayPorts
	// policy has the final say on non-loopback binds.
	bind := net.JoinHostPort(localBindOr(f.RemoteHost), strconv.Itoa(f.RemotePort))
	l, err := s.client.Listen("tcp", bind)
	if err != nil {
		s.emitTunnel(onTunnel, TunnelStatus{Kind: "remote", Name: f.Name, Listen: bind, OK: false,
			Err: fmt.Sprintf("remote forward: listen %s: %v (server may reject or require GatewayPorts)", bind, err)})
		return
	}
	if !s.tunnels.addListener(l) {
		_ = l.Close()
		return
	}
	s.emitTunnel(onTunnel, TunnelStatus{Kind: "remote", Name: f.Name, Listen: bind, OK: true})
	localTarget := net.JoinHostPort(localBindOr(f.LocalHost), strconv.Itoa(f.LocalPort))
	go s.acceptLoop(l, func(remote net.Conn) {
		local, derr := net.Dial("tcp", localTarget)
		if derr != nil {
			_ = remote.Close()
			return
		}
		s.pipe(remote, local)
	})
}

// acceptLoop accepts connections until the listener is closed, handing each to
// handle in its own goroutine.
func (s *Session) acceptLoop(l net.Listener, handle func(net.Conn)) {
	for {
		c, err := l.Accept()
		if err != nil {
			return // listener closed => tunnel torn down
		}
		go handle(c)
	}
}

// pipe copies bidirectionally between two connections, tracking both so a
// session teardown closes them. It returns when either direction ends.
func (s *Session) pipe(a, b net.Conn) {
	if !s.tunnels.track(a) || !s.tunnels.track(b) {
		// Session already closing; drop the connections.
		_ = a.Close()
		_ = b.Close()
		s.tunnels.untrack(a)
		s.tunnels.untrack(b)
		return
	}
	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		_ = a.Close()
		_ = b.Close()
		done <- struct{}{}
	}
	go cp(a, b)
	go cp(b, a)
	<-done
	<-done
	s.tunnels.untrack(a)
	s.tunnels.untrack(b)
}
