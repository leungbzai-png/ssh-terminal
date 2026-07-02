package session

import (
	"os"
	"strings"
	"testing"
)

func TestSaveLoadDropsQuickTabsAndSecrets(t *testing.T) {
	_ = os.Remove(path())

	// A Quick Connect tab has no hostId; its display name here stands in for
	// anything that must never be persisted.
	const quickMarker = "QUICK-NEVER-PERSIST-9f21"
	err := Save([]OpenTab{
		{HostID: "h1", HostName: "web"},
		{HostID: "", HostName: quickMarker},
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	got := Load()
	if len(got) != 1 || got[0].HostID != "h1" || got[0].HostName != "web" {
		t.Fatalf("expected only the saved-host tab, got %+v", got)
	}

	raw, err := os.ReadFile(path())
	if err != nil {
		t.Fatalf("read session.json: %v", err)
	}
	s := string(raw)
	if strings.Contains(s, quickMarker) {
		t.Fatal("Quick Connect tab (no hostId) must not be persisted")
	}
	for _, bad := range []string{"password", "passphrase", "PRIVATE KEY"} {
		if strings.Contains(s, bad) {
			t.Fatalf("session.json must not contain %q", bad)
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	_ = os.Remove(path())
	if got := Load(); len(got) != 0 {
		t.Fatalf("expected empty slice for missing file, got %+v", got)
	}
}
