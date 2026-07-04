//go:build integration
// +build integration

// Build-tagged, backend-live integration tests for the v0.9.0 Advanced SSH code
// paths. They drive the real *Manager against disposable in-process SSH servers
// on 127.0.0.1 (see integration_server_test.go) and are EXCLUDED from a normal
// `go test ./...`.
//
// Run them with:
//
//	go test -tags=integration ./...
//	go test -tags=integration ./internal/sshsess -run Integration -v
//
// Everything is localhost-only, needs no real server or secret, and generates
// all credentials at runtime. No password, private key, or PEM block is ever
// printed; error text is routed through internal/redact.
package sshsess

import (
	"errors"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/leungbzai-png/ssh-terminal/internal/hosts"
	"github.com/leungbzai-png/ssh-terminal/internal/redact"
)

// resetKnownHosts removes the per-process known_hosts file so every connection
// in a subtest is first-seen. Without this, an ephemeral port reused by a later
// testServer (with a different host key) would hit a knownhosts mismatch and
// hard-fail BEFORE the auto-accept prompt runs — a flaky poison for a gate.
func resetKnownHosts(t *testing.T) {
	t.Helper()
	_ = os.Remove(knownHostsPath())
}

// safeErr redacts secrets out of an error before it reaches the test log.
func safeErr(err error, secrets ...string) string {
	return redact.Error(err, secrets...)
}

// tunnelCollector records every TunnelStatus emitted for a session. Guarded by a
// mutex because emitTunnel fires from forward goroutines (a bare slice append
// would be a data race under -race).
type tunnelCollector struct {
	mu     sync.Mutex
	events []TunnelStatus
}

func (c *tunnelCollector) handler(_ string, st TunnelStatus) {
	c.mu.Lock()
	c.events = append(c.events, st)
	c.mu.Unlock()
}

// waitFor polls the collected events for one matching pred, up to d. Returns the
// matching status and true, or a zero status and false on timeout.
func (c *tunnelCollector) waitFor(pred func(TunnelStatus) bool, d time.Duration) (TunnelStatus, bool) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		for _, e := range c.events {
			if pred(e) {
				c.mu.Unlock()
				return e, true
			}
		}
		c.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	return TunnelStatus{}, false
}

// newManagerT builds a Manager that auto-accepts host keys and routes tunnel
// events into the returned collector. onClose may be nil.
func newManagerT(onClose CloseHandler, c *tunnelCollector) *Manager {
	tunnel := func(string, TunnelStatus) {}
	if c != nil {
		tunnel = c.handler
	}
	if onClose == nil {
		onClose = func(string, string) {}
	}
	return NewManager(
		func(string, []byte) {},
		onClose,
		func(string, string) bool { return true }, // auto-accept host key
		tunnel,
	)
}

// TestIntegrationAdvancedSSH exercises the Advanced SSH backend end to end
// against embedded localhost SSH servers. Subtests run sequentially (no
// t.Parallel) because they share the per-process known_hosts file.
func TestIntegrationAdvancedSSH(t *testing.T) {
	// A. ProxyJump / Bastion ------------------------------------------------
	t.Run("ProxyJumpBastion", func(t *testing.T) {
		resetKnownHosts(t)
		jump := newTestServer(t)
		target := newTestServer(t)
		mgr := newManagerT(nil, nil)

		th := hostFor(target)
		jh := hostFor(jump)
		if err := mgr.Open(OpenOptions{SessionID: "pj", Host: th, JumpHost: &jh, Cols: 80, Rows: 24, TimeoutSec: 5}); err != nil {
			t.Fatalf("ProxyJump open: %s", safeErr(err, th.Password, jh.Password))
		}
		if mgr.ActiveCount() != 1 {
			t.Fatalf("ActiveCount = %d, want 1", mgr.ActiveCount())
		}
		if err := mgr.Close("pj"); err != nil {
			t.Fatalf("close: %v", err)
		}
		if mgr.ActiveCount() != 0 {
			t.Fatalf("ActiveCount after close = %d, want 0", mgr.ActiveCount())
		}
	})

	// B. Local forwarding ---------------------------------------------------
	t.Run("LocalForward", func(t *testing.T) {
		resetKnownHosts(t)
		srv := newTestServer(t)
		col := &tunnelCollector{}
		mgr := newManagerT(nil, col)

		sentinel := "SENTINEL-LOCAL-" + randHex(4)
		svcPort := startHTTP(t, sentinel)
		lp := freePort(t)

		h := hostFor(srv)
		h.Advanced = &hosts.AdvancedSSH{LocalForwards: []hosts.Forward{{
			Name: "web", LocalHost: "", LocalPort: lp, RemoteHost: "127.0.0.1", RemotePort: svcPort, Enabled: true,
		}}}
		if err := mgr.Open(OpenOptions{SessionID: "lf", Host: h, Cols: 80, Rows: 24, TimeoutSec: 5}); err != nil {
			t.Fatalf("open: %s", safeErr(err, h.Password))
		}
		// Readiness: wait for the bind event, then probe with bounded retry.
		if _, ok := col.waitFor(func(st TunnelStatus) bool { return st.Kind == "local" && st.OK }, 3*time.Second); !ok {
			t.Fatalf("local forward never reported bound")
		}
		addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(lp))
		if !waitCond(func() bool { return httpGetVia(addr, sentinel) }, 3*time.Second) {
			t.Fatalf("local forward did not carry traffic on default 127.0.0.1 bind")
		}
		_ = mgr.Close("lf")
		if !waitPortFree(lp, 2*time.Second) {
			t.Fatalf("local listener not released after close (port %d)", lp)
		}
	})

	// B'. Occupied local port must not abort the session, and must surface a
	// readable, forward-classified error via the tunnel status.
	t.Run("LocalForwardOccupiedPort", func(t *testing.T) {
		resetKnownHosts(t)
		srv := newTestServer(t)
		col := &tunnelCollector{}
		mgr := newManagerT(nil, col)

		sentinel := "SENTINEL-OCC-" + randHex(4)
		svcPort := startHTTP(t, sentinel)
		occ := freePort(t)
		blocker, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(occ)))
		if err != nil {
			t.Fatalf("occupy port: %v", err)
		}
		defer blocker.Close()

		h := hostFor(srv)
		h.Advanced = &hosts.AdvancedSSH{LocalForwards: []hosts.Forward{{
			Name: "occupied", LocalHost: "127.0.0.1", LocalPort: occ, RemoteHost: "127.0.0.1", RemotePort: svcPort, Enabled: true,
		}}}
		if err := mgr.Open(OpenOptions{SessionID: "occ", Host: h, Cols: 80, Rows: 24, TimeoutSec: 5}); err != nil {
			t.Fatalf("session must still open despite bind failure: %s", safeErr(err, h.Password))
		}
		if mgr.ActiveCount() != 1 {
			t.Fatalf("ActiveCount = %d, want 1 (bind failure must not abort session)", mgr.ActiveCount())
		}
		st, ok := col.waitFor(func(st TunnelStatus) bool { return st.Kind == "local" && !st.OK }, 3*time.Second)
		if !ok {
			t.Fatalf("occupied-port forward never reported a failure status")
		}
		if st.Err == "" {
			t.Fatalf("failed tunnel status has empty Err")
		}
		if cat, _ := classifyError(errors.New(st.Err)); cat != DiagForward {
			t.Errorf("occupied-port Err classified as %q, want %q (err=%q)", cat, DiagForward, st.Err)
		}
		_ = mgr.Close("occ")
	})

	// C. Remote forwarding --------------------------------------------------
	t.Run("RemoteForward", func(t *testing.T) {
		resetKnownHosts(t)
		srv := newTestServer(t)
		col := &tunnelCollector{}
		mgr := newManagerT(nil, col)

		sentinel := "SENTINEL-REMOTE-" + randHex(4)
		svcPort := startHTTP(t, sentinel)
		rp := freePort(t)

		h := hostFor(srv)
		h.Advanced = &hosts.AdvancedSSH{RemoteForwards: []hosts.Forward{{
			Name: "back", RemoteHost: "127.0.0.1", RemotePort: rp, LocalHost: "127.0.0.1", LocalPort: svcPort, Enabled: true,
		}}}
		if err := mgr.Open(OpenOptions{SessionID: "rf", Host: h, Cols: 80, Rows: 24, TimeoutSec: 5}); err != nil {
			t.Fatalf("open: %s", safeErr(err, h.Password))
		}
		if _, ok := col.waitFor(func(st TunnelStatus) bool { return st.Kind == "remote" && st.OK }, 3*time.Second); !ok {
			t.Fatalf("remote forward never reported bound")
		}
		// The remote-forwarded port is bound on the (in-process) server side; we
		// reach it over loopback because the server binds 127.0.0.1:rp.
		addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(rp))
		if !waitCond(func() bool { return httpGetVia(addr, sentinel) }, 3*time.Second) {
			t.Fatalf("remote forward did not carry traffic")
		}
		_ = mgr.Close("rf")
		if !waitPortFree(rp, 2*time.Second) {
			t.Fatalf("remote forward listener not released after close (port %d)", rp)
		}
	})

	// D. Dynamic SOCKS5 -----------------------------------------------------
	t.Run("DynamicSOCKS", func(t *testing.T) {
		resetKnownHosts(t)
		srv := newTestServer(t)
		col := &tunnelCollector{}
		mgr := newManagerT(nil, col)

		sentinel := "SENTINEL-SOCKS-" + randHex(4)
		svcPort := startHTTP(t, sentinel)
		sp := freePort(t)

		h := hostFor(srv)
		h.Advanced = &hosts.AdvancedSSH{DynamicForwards: []hosts.DynamicForward{{
			Name: "socks", LocalHost: "", LocalPort: sp, Enabled: true, // default 127.0.0.1 bind
		}}}
		if err := mgr.Open(OpenOptions{SessionID: "dyn", Host: h, Cols: 80, Rows: 24, TimeoutSec: 5}); err != nil {
			t.Fatalf("open: %s", safeErr(err, h.Password))
		}
		if _, ok := col.waitFor(func(st TunnelStatus) bool { return st.Kind == "dynamic" && st.OK }, 3*time.Second); !ok {
			t.Fatalf("dynamic forward never reported bound")
		}
		proxy := net.JoinHostPort("127.0.0.1", strconv.Itoa(sp))
		ok := waitCond(func() bool {
			got, _ := socks5Get(proxy, "127.0.0.1", svcPort, sentinel)
			return got
		}, 3*time.Second)
		if !ok {
			// One more attempt to surface the underlying error text.
			if _, serr := socks5Get(proxy, "127.0.0.1", svcPort, sentinel); serr != nil {
				t.Fatalf("SOCKS5 proxy did not carry traffic: %s", safeErr(serr))
			}
			t.Fatalf("SOCKS5 proxy did not carry traffic")
		}
		_ = mgr.Close("dyn")
		if !waitPortFree(sp, 2*time.Second) {
			t.Fatalf("SOCKS listener not released after close (port %d)", sp)
		}
	})

	// E. Connection diagnostics --------------------------------------------
	t.Run("Diagnostics", func(t *testing.T) {
		resetKnownHosts(t)
		srv := newTestServer(t)
		mgr := newManagerT(nil, nil)

		assert := func(name string, opt OpenOptions, want DiagCategory, secrets ...string) {
			t.Helper()
			err := mgr.Open(opt)
			if err == nil {
				t.Errorf("%s: expected error, got nil", name)
				return
			}
			got, msg := classifyError(err)
			if got != want {
				t.Errorf("%s: classify = %q, want %q (err=%s)", name, got, want, safeErr(err, secrets...))
			}
			if msg == "" {
				t.Errorf("%s: empty diagnostic message", name)
			}
		}

		// TCP refused: a definitely-closed loopback port.
		refused := hostFor(srv)
		refused.Address = "127.0.0.1"
		refused.Port = freePort(t)
		assert("refused", OpenOptions{SessionID: "d-ref", Host: refused, TimeoutSec: 3}, DiagTCP, refused.Password)

		// Auth failure: wrong password.
		badPw := hostFor(srv)
		badPw.Password = "wrong-" + randHex(4)
		assert("auth", OpenOptions{SessionID: "d-auth", Host: badPw, TimeoutSec: 5}, DiagAuth, badPw.Password)

		// DNS failure: an unresolvable .invalid name.
		dns := hostFor(srv)
		dns.Address = "nonexistent-" + randHex(6) + ".invalid"
		dns.Port = 22
		assert("dns", OpenOptions{SessionID: "d-dns", Host: dns, TimeoutSec: 3}, DiagDNS, dns.Password)

		// Key/passphrase failure: a garbage key file at KeyPath.
		badKey := hostFor(srv)
		badKey.AuthType = "key"
		badKey.Password = ""
		badKey.KeyPath = writeGarbageKey(t)
		assert("key", OpenOptions{SessionID: "d-key", Host: badKey, TimeoutSec: 5}, DiagKey)

		// ProxyJump failure: bastion at a closed loopback port.
		pjTarget := hostFor(srv)
		pjJump := hostFor(srv)
		pjJump.Address = "127.0.0.1"
		pjJump.Port = freePort(t)
		assert("proxyjump", OpenOptions{SessionID: "d-pj", Host: pjTarget, JumpHost: &pjJump, TimeoutSec: 3}, DiagProxyJump, pjTarget.Password, pjJump.Password)
	})

	// G. Auto-reconnect backend signal -------------------------------------
	// The frontend cap/cancel/discriminator logic is Vue-only and NOT covered
	// here; this asserts only the backend close signal it keys on.
	t.Run("ReconnectCloseSignal", func(t *testing.T) {
		resetKnownHosts(t)

		// Unexpected drop => a non-user close reason (reconnect-eligible).
		t.Run("UnexpectedDrop", func(t *testing.T) {
			resetKnownHosts(t)
			srv := newTestServer(t)
			closeCh := make(chan string, 1)
			mgr := newManagerT(func(_ string, reason string) {
				select {
				case closeCh <- reason:
				default:
				}
			}, nil)
			h := hostFor(srv)
			if err := mgr.Open(OpenOptions{SessionID: "drop", Host: h, Cols: 80, Rows: 24, TimeoutSec: 5}); err != nil {
				t.Fatalf("open: %s", safeErr(err, h.Password))
			}
			// Ensure the session is fully wired before we drop it.
			if !waitCond(func() bool { return mgr.ActiveCount() == 1 }, 2*time.Second) {
				t.Fatalf("session did not become active")
			}
			srv.dropAll() // simulate an unexpected network drop
			var reason string
			select {
			case reason = <-closeCh:
			case <-time.After(3 * time.Second):
				t.Fatalf("no close signal after unexpected drop")
			}
			if reason == "" || reason == "user closed" {
				t.Fatalf("drop reason = %q, want a non-empty non-user reason", reason)
			}
		})

		// User-initiated close => the distinguishable "user closed" reason.
		t.Run("UserClose", func(t *testing.T) {
			resetKnownHosts(t)
			srv := newTestServer(t)
			closeCh := make(chan string, 1)
			mgr := newManagerT(func(_ string, reason string) {
				select {
				case closeCh <- reason:
				default:
				}
			}, nil)
			h := hostFor(srv)
			if err := mgr.Open(OpenOptions{SessionID: "uc", Host: h, Cols: 80, Rows: 24, TimeoutSec: 5}); err != nil {
				t.Fatalf("open: %s", safeErr(err, h.Password))
			}
			_ = mgr.Close("uc")
			select {
			case reason := <-closeCh:
				if reason != "user closed" {
					t.Fatalf("user close reason = %q, want %q", reason, "user closed")
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("no close signal after user close")
			}
		})
	})

	// F. Runtime cleanup ----------------------------------------------------
	// Per-tunnel listener release is asserted in B/C/D above; here we confirm
	// CloseAll drains the active-session count to zero.
	t.Run("CleanupCloseAll", func(t *testing.T) {
		resetKnownHosts(t)
		srv := newTestServer(t)
		mgr := newManagerT(nil, nil)
		for _, id := range []string{"s1", "s2", "s3"} {
			if err := mgr.Open(OpenOptions{SessionID: id, Host: hostFor(srv), Cols: 80, Rows: 24, TimeoutSec: 5}); err != nil {
				t.Fatalf("open %s: %v", id, err)
			}
		}
		if mgr.ActiveCount() != 3 {
			t.Fatalf("ActiveCount = %d, want 3", mgr.ActiveCount())
		}
		mgr.CloseAll()
		if mgr.ActiveCount() != 0 {
			t.Fatalf("ActiveCount after CloseAll = %d, want 0", mgr.ActiveCount())
		}
	})
}

// writeGarbageKey writes a non-parseable "private key" file into t.TempDir and
// returns its path. No real key material is used.
func writeGarbageKey(t *testing.T) string {
	t.Helper()
	path := t.TempDir() + "/id_bad"
	// Deliberately malformed so ssh.ParsePrivateKey fails with "parse key".
	if err := os.WriteFile(path, []byte("not-a-real-key\n"), 0o600); err != nil {
		t.Fatalf("write garbage key: %v", err)
	}
	return path
}
