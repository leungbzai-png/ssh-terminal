package hosts

import (
	"os"
	"strings"
	"testing"

	"github.com/leungbzai-png/ssh-terminal/internal/portable"
)

// resetCache clears the in-memory host cache so each test starts clean without
// reading whatever hosts.json a previous test left in the temp data dir.
func resetCache() {
	mu.Lock()
	cache = map[string]storedHost{}
	mu.Unlock()
}

// pemHeaders are private-key markers that must never appear in an export.
var pemHeaders = []string{
	"PRIVATE KEY",
	"BEGIN OPENSSH PRIVATE KEY",
	"BEGIN RSA PRIVATE KEY",
	"BEGIN EC PRIVATE KEY",
	"BEGIN DSA PRIVATE KEY",
}

func TestBuildExportExcludesSecrets(t *testing.T) {
	resetCache()
	const pwSentinel = "SENTINEL-PW-9f3a2b71"
	const ppSentinel = "SENTINEL-PP-c4d5e6f7"

	if _, err := Upsert(Host{
		Name: "web", Address: "host.example.test", Port: 22, User: "deploy",
		AuthType: "password", Password: pwSentinel, Passphrase: ppSentinel,
		Group: "Production", Note: "primary",
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	data, err := MarshalExport()
	if err != nil {
		t.Fatalf("MarshalExport: %v", err)
	}
	s := string(data)

	// The unique sentinels can never legitimately appear — an exact-match guard.
	if strings.Contains(s, pwSentinel) {
		t.Fatal("export leaked plaintext password")
	}
	if strings.Contains(s, ppSentinel) {
		t.Fatal("export leaked plaintext passphrase")
	}
	// No encrypted-secret field names. (Note: the literal substring "password"
	// legitimately appears as the authType value "password", so we assert on the
	// exact JSON key form and on the unique sentinels above, not on "password".)
	for _, bad := range []string{"encPassword", "encPassphrase", "\"passphrase\":"} {
		if strings.Contains(s, bad) {
			t.Fatalf("export contains forbidden field %q", bad)
		}
	}
	for _, h := range pemHeaders {
		if strings.Contains(s, h) {
			t.Fatalf("export contains private-key marker %q", h)
		}
	}
	// Non-secret metadata must survive.
	for _, want := range []string{"host.example.test", "deploy", "Production", ExportFormat} {
		if !strings.Contains(s, want) {
			t.Fatalf("export missing expected field %q", want)
		}
	}
}

func TestParseExportValidation(t *testing.T) {
	if _, err := ParseExport([]byte("not json at all")); err == nil {
		t.Error("expected error for invalid JSON")
	}
	if _, err := ParseExport([]byte(`{"format":"something.else","version":1}`)); err == nil {
		t.Error("expected error for unknown format")
	}
	if _, err := ParseExport([]byte(`{"format":"` + ExportFormat + `","version":999}`)); err == nil {
		t.Error("expected error for unsupported version")
	}
	good := `{"format":"` + ExportFormat + `","version":1,"hosts":[{"name":"a","address":"h","port":22,"user":"u","authType":"password"}]}`
	exp, err := ParseExport([]byte(good))
	if err != nil {
		t.Fatalf("ParseExport(valid): %v", err)
	}
	if len(exp.Hosts) != 1 || exp.Hosts[0].Address != "h" {
		t.Fatalf("unexpected parse result: %+v", exp)
	}
}

func TestHostsJsonNoPlaintextSecretOnDisk(t *testing.T) {
	resetCache()
	const sentinel = "SENTINEL-DISK-a1b2c3d4"
	if _, err := Upsert(Host{
		Name: "db", Address: "db.example.test", Port: 2222, User: "root",
		AuthType: "password", Password: sentinel, Passphrase: sentinel + "-pp",
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	raw, err := os.ReadFile(portable.DataPath("hosts.json"))
	if err != nil {
		t.Fatalf("read hosts.json: %v", err)
	}
	if strings.Contains(string(raw), sentinel) {
		t.Fatal("hosts.json contains plaintext secret")
	}
	// The encrypted fields must be present (proves secrets are stored, encrypted).
	if !strings.Contains(string(raw), "encPassword") {
		t.Fatal("expected encPassword field in hosts.json")
	}
}
