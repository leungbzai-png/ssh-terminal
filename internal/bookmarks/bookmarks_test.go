package bookmarks

import (
	"os"
	"strings"
	"testing"

	"github.com/leungbzai-png/ssh-terminal/internal/portable"
)

func reset(t *testing.T) {
	t.Helper()
	mu.Lock()
	cache = nil
	loaded = false
	mu.Unlock()
	_ = os.Remove(filePath())
}

func TestBookmarksCRUDAndScoping(t *testing.T) {
	reset(t)

	b, err := Add("host1", "web root", "/var/www")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if b.ID == "" || b.HostID != "host1" || b.Path != "/var/www" {
		t.Fatalf("unexpected bookmark: %+v", b)
	}

	// Idempotent: same host+path returns the existing record.
	b2, err := Add("host1", "different name", "/var/www")
	if err != nil {
		t.Fatalf("Add dup: %v", err)
	}
	if b2.ID != b.ID {
		t.Fatal("duplicate host+path should return existing bookmark")
	}

	// Host scoping.
	if _, err := Add("host2", "opt", "/opt"); err != nil {
		t.Fatalf("Add host2: %v", err)
	}
	if l, _ := List("host1"); len(l) != 1 {
		t.Fatalf("host1 should have 1 bookmark, got %d", len(l))
	}
	if l, _ := List("host2"); len(l) != 1 {
		t.Fatalf("host2 should have 1 bookmark, got %d", len(l))
	}
	if l, _ := List(""); len(l) != 0 {
		t.Fatalf("empty hostID should return 0 bookmarks, got %d", len(l))
	}

	// Delete.
	if err := Delete(b.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if l, _ := List("host1"); len(l) != 0 {
		t.Fatalf("host1 should be empty after delete, got %d", len(l))
	}

	// Validation.
	if _, err := Add("", "n", "/p"); err == nil {
		t.Error("expected error for empty hostID")
	}
	if _, err := Add("h", "n", ""); err == nil {
		t.Error("expected error for empty path")
	}
}

func TestBookmarksNoSecretsOnDisk(t *testing.T) {
	reset(t)
	if _, err := Add("hostX", "home", "/home/user/project"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	raw, err := os.ReadFile(filePath())
	if err != nil {
		t.Fatalf("read bookmarks.json: %v", err)
	}
	s := string(raw)
	// Only whitelisted, non-secret fields.
	for _, want := range []string{"hostId", "name", "path"} {
		if !strings.Contains(s, want) {
			t.Fatalf("expected field %q in bookmarks.json", want)
		}
	}
	for _, bad := range []string{"password", "passphrase", "PRIVATE KEY", "encPassword"} {
		if strings.Contains(s, bad) {
			t.Fatalf("bookmarks.json must not contain %q", bad)
		}
	}
	_ = portable.DataDir() // ensure data dir resolved
}
