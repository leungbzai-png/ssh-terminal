package hosts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAdvancedNormalizeDefaultsLocalBind(t *testing.T) {
	a := &AdvancedSSH{
		LocalForwards:   []Forward{{Name: "web", LocalPort: 8080, RemoteHost: "127.0.0.1", RemotePort: 80, Enabled: true}},
		DynamicForwards: []DynamicForward{{Name: "socks", LocalPort: 1080, Enabled: true}},
	}
	if err := a.Normalize(); err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if a.LocalForwards[0].LocalHost != defaultLocalBind {
		t.Errorf("local forward bind = %q, want %q", a.LocalForwards[0].LocalHost, defaultLocalBind)
	}
	if a.DynamicForwards[0].LocalHost != defaultLocalBind {
		t.Errorf("dynamic forward bind = %q, want %q", a.DynamicForwards[0].LocalHost, defaultLocalBind)
	}
}

func TestAdvancedNormalizeNilSafe(t *testing.T) {
	var a *AdvancedSSH
	if err := a.Normalize(); err != nil {
		t.Fatalf("nil Normalize should be a no-op, got %v", err)
	}
}

func TestAdvancedNormalizeRejectsBadPorts(t *testing.T) {
	cases := []struct {
		name string
		a    *AdvancedSSH
	}{
		{"local port 0", &AdvancedSSH{LocalForwards: []Forward{{LocalPort: 0, RemoteHost: "h", RemotePort: 80, Enabled: true}}}},
		{"local port too big", &AdvancedSSH{LocalForwards: []Forward{{LocalPort: 70000, RemoteHost: "h", RemotePort: 80, Enabled: true}}}},
		{"remote missing host", &AdvancedSSH{LocalForwards: []Forward{{LocalPort: 8080, RemotePort: 80, Enabled: true}}}},
		{"dynamic bad port", &AdvancedSSH{DynamicForwards: []DynamicForward{{LocalPort: -1, Enabled: true}}}},
		{"remote fwd bad port", &AdvancedSSH{RemoteForwards: []Forward{{RemotePort: 0, LocalPort: 3000, Enabled: true}}}},
	}
	for _, c := range cases {
		if err := c.a.Normalize(); err == nil {
			t.Errorf("%s: expected validation error, got nil", c.name)
		}
	}
}

func TestAdvancedNormalizeDisabledDraftAllowed(t *testing.T) {
	// A disabled, half-filled forward must not block a save.
	a := &AdvancedSSH{LocalForwards: []Forward{{Name: "draft", Enabled: false}}}
	if err := a.Normalize(); err != nil {
		t.Fatalf("disabled draft should be allowed: %v", err)
	}
}

func TestAdvancedNormalizeDuplicateLocalBind(t *testing.T) {
	a := &AdvancedSSH{
		LocalForwards:   []Forward{{Name: "a", LocalHost: "127.0.0.1", LocalPort: 9000, RemoteHost: "h", RemotePort: 80, Enabled: true}},
		DynamicForwards: []DynamicForward{{Name: "b", LocalHost: "127.0.0.1", LocalPort: 9000, Enabled: true}},
	}
	if err := a.Normalize(); err == nil {
		t.Fatal("expected duplicate local bind error")
	}
}

func TestProxyJumpManualRequiresAddressAndUser(t *testing.T) {
	a := &AdvancedSSH{ProxyJump: &ProxyJump{Mode: ProxyJumpManual, Address: "", User: "u"}}
	if err := a.Normalize(); err == nil {
		t.Error("manual proxy jump without address should error")
	}
	a = &AdvancedSSH{ProxyJump: &ProxyJump{Mode: ProxyJumpManual, Address: "bastion", User: ""}}
	if err := a.Normalize(); err == nil {
		t.Error("manual proxy jump without user should error")
	}
	a = &AdvancedSSH{ProxyJump: &ProxyJump{Mode: ProxyJumpManual, Address: "bastion", User: "u"}}
	if err := a.Normalize(); err != nil {
		t.Fatalf("valid manual proxy jump: %v", err)
	}
	if a.ProxyJump.Port != 22 {
		t.Errorf("manual proxy jump default port = %d, want 22", a.ProxyJump.Port)
	}
}

func TestProxyJumpSavedHostRequiresID(t *testing.T) {
	a := &AdvancedSSH{ProxyJump: &ProxyJump{Mode: ProxyJumpSavedHost, JumpHostID: ""}}
	if err := a.Normalize(); err == nil {
		t.Error("savedHost proxy jump without id should error")
	}
}

func TestProxyJumpUnknownMode(t *testing.T) {
	a := &AdvancedSSH{ProxyJump: &ProxyJump{Mode: "bogus"}}
	if err := a.Normalize(); err == nil {
		t.Error("unknown proxy jump mode should error")
	}
}

// A manual bastion can never carry a secret: the marshalled AdvancedSSH must
// contain no password/passphrase/secret keys. This guards against someone
// accidentally adding a secret field to ProxyJump/Forward in the future.
func TestAdvancedMarshalHasNoSecretKeys(t *testing.T) {
	a := &AdvancedSSH{
		ProxyJump:       &ProxyJump{Mode: ProxyJumpManual, Address: "b", User: "u", KeyPath: "/k", Port: 22},
		LocalForwards:   []Forward{{Name: "web", LocalPort: 8080, RemoteHost: "127.0.0.1", RemotePort: 80, Enabled: true}},
		DynamicForwards: []DynamicForward{{Name: "socks", LocalPort: 1080, Enabled: true}},
		AutoReconnect:   &AutoReconnect{Enabled: true},
	}
	if err := a.Normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	blob, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	low := strings.ToLower(string(blob))
	for _, bad := range []string{"password", "passphrase", "\"secret", "encpassword", "encpassphrase"} {
		if strings.Contains(low, bad) {
			t.Fatalf("advanced config marshalled a secret-looking key %q: %s", bad, blob)
		}
	}
}

func TestAutoReconnectNormalizeClamps(t *testing.T) {
	r := &AutoReconnect{Enabled: true, MaxAttempts: 0, DelaySeconds: 0}
	a := &AdvancedSSH{AutoReconnect: r}
	if err := a.Normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if r.MaxAttempts != defaultReconnectMax {
		t.Errorf("MaxAttempts = %d, want default %d", r.MaxAttempts, defaultReconnectMax)
	}
	if r.DelaySeconds != defaultReconnectDelay {
		t.Errorf("DelaySeconds = %d, want default %d", r.DelaySeconds, defaultReconnectDelay)
	}

	r2 := &AutoReconnect{Enabled: true, MaxAttempts: 999, DelaySeconds: 999}
	a2 := &AdvancedSSH{AutoReconnect: r2}
	if err := a2.Normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if r2.MaxAttempts != maxReconnectAttempts {
		t.Errorf("MaxAttempts = %d, want clamp %d", r2.MaxAttempts, maxReconnectAttempts)
	}
	if r2.DelaySeconds != maxReconnectDelaySecond {
		t.Errorf("DelaySeconds = %d, want clamp %d", r2.DelaySeconds, maxReconnectDelaySecond)
	}
}

func TestIsWildcardBind(t *testing.T) {
	for _, w := range []string{"0.0.0.0", "::", "*"} {
		if !IsWildcardBind(w) {
			t.Errorf("IsWildcardBind(%q) = false, want true", w)
		}
	}
	for _, ok := range []string{"127.0.0.1", "localhost", "10.0.0.5"} {
		if IsWildcardBind(ok) {
			t.Errorf("IsWildcardBind(%q) = true, want false", ok)
		}
	}
}
