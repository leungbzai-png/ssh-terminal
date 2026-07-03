package hosts

import (
	"os"
	"strings"
	"testing"

	"github.com/leungbzai-png/ssh-terminal/internal/portable"
)

// writeHostsJSON writes raw bytes to the data-dir hosts.json and resets the
// in-memory cache so the next access re-reads from disk.
func writeHostsJSON(t *testing.T, raw string) {
	t.Helper()
	if err := os.WriteFile(portable.DataPath(file), []byte(raw), 0o600); err != nil {
		t.Fatalf("write hosts.json: %v", err)
	}
	mu.Lock()
	cache = nil
	mu.Unlock()
}

// A v0.7.0-style hosts.json (no "advanced" field) must load unchanged.
func TestLoadV070HostNoAdvanced(t *testing.T) {
	writeHostsJSON(t, `[
	  {"id":"a1","name":"legacy","address":"h.example","port":22,"user":"root","authType":"password","encPassword":"deadbeef","updatedAt":1}
	]`)
	list, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 host, got %d", len(list))
	}
	if list[0].Advanced != nil {
		t.Errorf("legacy host should have nil Advanced, got %+v", list[0].Advanced)
	}
	if list[0].Address != "h.example" {
		t.Errorf("address = %q", list[0].Address)
	}
}

// Missing optional fields and unknown extra fields must not crash the loader.
func TestLoadMissingAndUnknownFields(t *testing.T) {
	writeHostsJSON(t, `[
	  {"id":"b1","address":"only-address","futureField":{"x":1},"anotherUnknown":[1,2,3]}
	]`)
	list, err := List()
	if err != nil {
		t.Fatalf("List with unknown fields: %v", err)
	}
	if len(list) != 1 || list[0].Address != "only-address" {
		t.Fatalf("unexpected list: %+v", list)
	}
	// Missing port/user default sanely; no panic is the key assertion.
}

// Corrupt JSON must return an error (not panic) and must NOT silently empty the
// file — a subsequent load keeps failing until the file is fixed.
func TestLoadCorruptJSONErrorsNoPanic(t *testing.T) {
	writeHostsJSON(t, `{ this is not valid json ][`)
	if _, err := List(); err == nil {
		t.Fatal("expected error for corrupt hosts.json")
	}
	// Second call must also error (cache flag was cleared), proving we didn't
	// mark a corrupt file as "loaded empty".
	if _, err := List(); err == nil {
		t.Fatal("expected persistent error for corrupt hosts.json")
	}
	// The on-disk file must be untouched (not overwritten with []).
	raw, _ := os.ReadFile(portable.DataPath(file))
	if strings.TrimSpace(string(raw)) == "[]" {
		t.Fatal("corrupt hosts.json was overwritten with empty list")
	}
}

// A host carrying valid Advanced config round-trips through save + reload.
func TestUpsertAdvancedRoundTrip(t *testing.T) {
	resetCache()
	_ = os.Remove(portable.DataPath(file))
	if _, err := Upsert(Host{
		Name: "adv", Address: "adv.example", Port: 22, User: "root", AuthType: "password",
		Advanced: &AdvancedSSH{
			LocalForwards: []Forward{{Name: "web", LocalPort: 8080, RemoteHost: "127.0.0.1", RemotePort: 80, Enabled: true}},
			ProxyJump:     &ProxyJump{Mode: ProxyJumpSavedHost, JumpHostID: "jh1"},
		},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	mu.Lock()
	cache = nil
	mu.Unlock()
	list, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Advanced == nil {
		t.Fatalf("advanced not persisted: %+v", list)
	}
	if list[0].Advanced.LocalForwards[0].LocalHost != defaultLocalBind {
		t.Errorf("local bind not defaulted on save: %q", list[0].Advanced.LocalForwards[0].LocalHost)
	}
}

// Invalid Advanced config must be rejected at Upsert (never persisted).
func TestUpsertRejectsInvalidAdvanced(t *testing.T) {
	resetCache()
	if _, err := Upsert(Host{
		Name: "bad", Address: "bad.example", User: "root", AuthType: "password",
		Advanced: &AdvancedSSH{LocalForwards: []Forward{{LocalPort: 999999, RemoteHost: "h", RemotePort: 80, Enabled: true}}},
	}); err == nil {
		t.Fatal("expected Upsert to reject invalid advanced config")
	}
}
