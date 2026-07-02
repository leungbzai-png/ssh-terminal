package sshconfig

import (
	"strings"
	"testing"
)

func findEntry(entries []Entry, alias string) *Entry {
	for i := range entries {
		if entries[i].Alias == alias {
			return &entries[i]
		}
	}
	return nil
}

func TestParseBasic(t *testing.T) {
	cfg := `
# a comment
Host web
    HostName 10.0.0.5
    User deploy
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
`
	entries, err := Parse(strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Alias != "web" || e.HostName != "10.0.0.5" || e.User != "deploy" || e.Port != 2222 {
		t.Fatalf("unexpected entry: %+v", e)
	}
	if e.IdentityFile != "~/.ssh/id_ed25519" {
		t.Fatalf("IdentityFile not preserved raw: %q", e.IdentityFile)
	}
}

func TestParseCaseInsensitiveAndEquals(t *testing.T) {
	cfg := `
HOST prod
  hostname=example.com
  USER = root
  PoRt=22
`
	entries, err := Parse(strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(entries, "prod")
	if e == nil {
		t.Fatalf("prod not found: %+v", entries)
	}
	if e.HostName != "example.com" || e.User != "root" || e.Port != 22 {
		t.Fatalf("case/equals parsing failed: %+v", *e)
	}
}

func TestParseMultiPatternUsesFirstConcrete(t *testing.T) {
	cfg := `
Host * !nope web1 web2
  HostName shared.example.com
`
	entries, err := Parse(strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Alias != "web1" {
		t.Fatalf("want single entry aliased web1, got %+v", entries)
	}
}

func TestParseSkipsWildcardOnlyBlock(t *testing.T) {
	cfg := `
Host *
  User globaluser
  IdentityFile ~/.ssh/global

Host real
  HostName r.example.com
`
	entries, err := Parse(strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Alias != "real" {
		t.Fatalf("wildcard block leaked: %+v", entries)
	}
	// Global directives must not bleed into the real block.
	if entries[0].User != "" || entries[0].IdentityFile != "" {
		t.Fatalf("global directives leaked into real block: %+v", entries[0])
	}
}

func TestParseMissingHostNameDefaultsToAlias(t *testing.T) {
	cfg := `
Host bare
  User me
`
	entries, err := Parse(strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(entries, "bare")
	if e == nil || e.HostName != "bare" {
		t.Fatalf("HostName should default to alias: %+v", entries)
	}
	if len(e.Warnings) == 0 {
		t.Fatalf("expected a warning about defaulted HostName")
	}
}

func TestParseUnsupportedDirectiveWarns(t *testing.T) {
	cfg := `
Host jump
  HostName j.example.com
  ProxyJump bastion
  LocalForward 8080 localhost:80
`
	entries, err := Parse(strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(entries, "jump")
	if e == nil {
		t.Fatal("jump not found")
	}
	if len(e.Warnings) < 2 {
		t.Fatalf("expected warnings for ProxyJump + LocalForward, got %v", e.Warnings)
	}
}
