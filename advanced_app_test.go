package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/leungbzai-png/ssh-terminal/internal/hosts"
)

// A connection error must never echo the host password/passphrase back to the
// UI, even if the underlying error text happened to contain it.
func TestConnectErrorRedactsSecrets(t *testing.T) {
	a := NewApp()
	h := &hosts.Host{Password: "SENTINEL-CONN-PW-88af21", Passphrase: "SENTINEL-CONN-PP-77be"}
	raw := errors.New("dial: ssh: handshake failed with password SENTINEL-CONN-PW-88af21 and passphrase SENTINEL-CONN-PP-77be")
	out := a.connectError(raw, h, nil).Error()
	if strings.Contains(out, "SENTINEL-CONN-PW-88af21") {
		t.Fatalf("connect error leaked password: %q", out)
	}
	if strings.Contains(out, "SENTINEL-CONN-PP-77be") {
		t.Fatalf("connect error leaked passphrase: %q", out)
	}
	// A readable diagnostic category prefix should still be present.
	if !strings.Contains(out, "：") {
		t.Fatalf("expected a diagnostic category prefix, got %q", out)
	}
}

// A jump-host password must also be scrubbed from a connect error.
func TestConnectErrorRedactsJumpSecret(t *testing.T) {
	a := NewApp()
	h := &hosts.Host{}
	jump := &hosts.Host{Password: "SENTINEL-JUMP-PW-abc123"}
	raw := errors.New("proxy jump: dial bastion: auth failed for SENTINEL-JUMP-PW-abc123")
	out := a.connectError(raw, h, jump).Error()
	if strings.Contains(out, "SENTINEL-JUMP-PW-abc123") {
		t.Fatalf("connect error leaked jump password: %q", out)
	}
}

// resolveJumpHost: manual mode without a key path is rejected (a manual bastion
// can never carry a password — it must be key-only).
func TestResolveJumpHostManualRequiresKey(t *testing.T) {
	a := NewApp()
	h := hosts.Host{Advanced: &hosts.AdvancedSSH{ProxyJump: &hosts.ProxyJump{
		Mode: hosts.ProxyJumpManual, Address: "bastion", User: "jump",
	}}}
	if _, err := a.resolveJumpHost(h); err == nil {
		t.Fatal("manual bastion without key should be rejected")
	}

	h.Advanced.ProxyJump.KeyPath = "/home/u/.ssh/id_ed25519"
	jump, err := a.resolveJumpHost(h)
	if err != nil {
		t.Fatalf("manual bastion with key: %v", err)
	}
	if jump == nil || jump.AuthType != "key" || jump.KeyPath == "" {
		t.Fatalf("unexpected resolved jump host: %+v", jump)
	}
	if jump.Password != "" || jump.Passphrase != "" {
		t.Fatal("manual bastion must have no secret")
	}
}

// resolveJumpHost: a savedHost reference to a missing host degrades to a clear
// error (the target host doesn't crash).
func TestResolveJumpHostSavedMissing(t *testing.T) {
	a := NewApp()
	h := hosts.Host{Advanced: &hosts.AdvancedSSH{ProxyJump: &hosts.ProxyJump{
		Mode: hosts.ProxyJumpSavedHost, JumpHostID: "does-not-exist-xyz",
	}}}
	if _, err := a.resolveJumpHost(h); err == nil {
		t.Fatal("missing saved jump host should error, not panic")
	}
}

// resolveJumpHost: no ProxyJump configured => (nil, nil).
func TestResolveJumpHostNone(t *testing.T) {
	a := NewApp()
	jump, err := a.resolveJumpHost(hosts.Host{})
	if err != nil || jump != nil {
		t.Fatalf("expected (nil,nil), got (%v,%v)", jump, err)
	}
}

// A safe export of a host that both stores a password AND references a saved
// jump host must not leak any secret, and must carry the (non-secret) advanced
// config so it round-trips.
func TestExportWithAdvancedHasNoSecret(t *testing.T) {
	// Clean slate for the shared host store.
	for _, h := range mustList(t) {
		_ = hosts.Delete(h.ID)
	}
	const pw = "SENTINEL-ADV-EXPORT-PW-5521"
	if _, err := hosts.Upsert(hosts.Host{
		Name: "target", Address: "t.example", Port: 22, User: "root",
		AuthType: "password", Password: pw,
		Advanced: &hosts.AdvancedSSH{
			ProxyJump:     &hosts.ProxyJump{Mode: hosts.ProxyJumpSavedHost, JumpHostID: "jh-id"},
			LocalForwards: []hosts.Forward{{Name: "web", LocalPort: 8080, RemoteHost: "127.0.0.1", RemotePort: 80, Enabled: true}},
		},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	data, err := hosts.MarshalExport()
	if err != nil {
		t.Fatalf("MarshalExport: %v", err)
	}
	s := string(data)
	if strings.Contains(s, pw) {
		t.Fatalf("advanced export leaked password")
	}
	for _, bad := range []string{"encPassword", "encPassphrase", "\"password\":", "\"passphrase\":"} {
		if strings.Contains(s, bad) {
			t.Fatalf("advanced export contains forbidden field %q", bad)
		}
	}
	// Non-secret advanced metadata must be present.
	if !strings.Contains(s, "proxyJump") || !strings.Contains(s, "localForwards") {
		t.Fatalf("advanced config missing from export: %s", s)
	}
}

func mustList(t *testing.T) []hosts.Host {
	t.Helper()
	list, err := hosts.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return list
}
