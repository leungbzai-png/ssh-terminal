package main

import (
	"testing"

	"github.com/leungbzai-png/ssh-terminal/internal/hosts"
)

// TestImportHostsDeduplicatesAndPreservesSecret exercises the import
// orchestration: duplicates skip by default, new hosts get fresh IDs, and
// overwriting an existing host preserves its encrypted password.
func TestImportHostsDeduplicatesAndPreservesSecret(t *testing.T) {
	a := NewApp()

	const pw = "SENTINEL-IMPORT-PW-4417"
	seed, err := hosts.Upsert(hosts.Host{
		Name: "existing", Address: "dup.example.test", Port: 22, User: "root",
		AuthType: "password", Password: pw,
	})
	if err != nil {
		t.Fatalf("seed Upsert: %v", err)
	}

	// An incoming host that duplicates the seed (address+port+user), carrying a
	// stale ID that must be ignored, plus a brand-new host.
	dup := hosts.SafeHost{
		Name: "existing-imported", Address: "dup.example.test", Port: 22,
		User: "root", AuthType: "password",
	}
	fresh := hosts.SafeHost{
		Name: "brand-new", Address: "new.example.test", Port: 2200,
		User: "deploy", AuthType: "password", Group: "Imported",
	}

	// overwrite=false: duplicate skipped, new host imported.
	res, err := a.ImportHosts([]hosts.SafeHost{dup, fresh}, false)
	if err != nil {
		t.Fatalf("ImportHosts: %v", err)
	}
	if res.Skipped != 1 || res.Imported != 1 || res.Overwritten != 0 {
		t.Fatalf("unexpected result (want skip=1 imported=1 overwrite=0): %+v", res)
	}

	// The new host received a fresh, non-empty ID distinct from the seed.
	list, err := hosts.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var newID string
	for _, h := range list {
		if h.Address == "new.example.test" {
			newID = h.ID
		}
	}
	if newID == "" {
		t.Fatal("new host was not imported / has no ID")
	}
	if newID == seed.ID {
		t.Fatal("imported host reused an existing ID instead of a fresh one")
	}

	// The skipped duplicate must not have altered the seed's stored password.
	got, err := hosts.Get(seed.ID)
	if err != nil {
		t.Fatalf("Get seed: %v", err)
	}
	if got.Password != pw {
		t.Fatal("seed password lost after skip-import")
	}

	// overwrite=true: duplicate updates the existing record in place and the
	// encrypted password is preserved (import carries no secret).
	res2, err := a.ImportHosts([]hosts.SafeHost{dup}, true)
	if err != nil {
		t.Fatalf("overwrite ImportHosts: %v", err)
	}
	if res2.Overwritten != 1 || res2.Imported != 0 || res2.Skipped != 0 {
		t.Fatalf("unexpected overwrite result (want overwrite=1): %+v", res2)
	}
	got2, err := hosts.Get(seed.ID)
	if err != nil {
		t.Fatalf("Get after overwrite: %v", err)
	}
	if got2.Password != pw {
		t.Fatal("overwrite wiped the stored password (must be preserved)")
	}
	if got2.Name != "existing-imported" {
		t.Fatalf("overwrite did not update metadata, name=%q", got2.Name)
	}
}
