package redact

import (
	"errors"
	"strings"
	"testing"
)

func TestStringRedactsSecretValue(t *testing.T) {
	const pw = "SUPER-SECRET-PW-91af"
	in := "dial failed for user root with password " + pw + " (auth error)"
	out := String(in, pw)
	if strings.Contains(out, pw) {
		t.Fatalf("secret leaked: %q", out)
	}
	if !strings.Contains(out, Placeholder) {
		t.Fatalf("expected placeholder in %q", out)
	}
	// Non-secret context must survive.
	if !strings.Contains(out, "auth error") {
		t.Fatalf("redaction destroyed non-secret text: %q", out)
	}
}

func TestStringRedactsMultipleSecrets(t *testing.T) {
	out := String("pw=ABCDEFGH pp=IJKLMNOP", "ABCDEFGH", "IJKLMNOP")
	if strings.Contains(out, "ABCDEFGH") || strings.Contains(out, "IJKLMNOP") {
		t.Fatalf("secret leaked: %q", out)
	}
}

func TestStringIgnoresShortSecrets(t *testing.T) {
	// A 1-char "secret" must not turn every 'a' into ***.
	out := String("banana", "a")
	if out != "banana" {
		t.Fatalf("short secret should be ignored, got %q", out)
	}
}

func TestStringIgnoresEmptySecret(t *testing.T) {
	out := String("hello world", "")
	if out != "hello world" {
		t.Fatalf("empty secret should be a no-op, got %q", out)
	}
}

func TestStringStripsPEMBlock(t *testing.T) {
	in := "error: -----BEGIN OPENSSH PRIVATE KEY-----\nabc123\ndef456\n-----END OPENSSH PRIVATE KEY----- trailing"
	out := String(in)
	if strings.Contains(out, "abc123") || strings.Contains(out, "BEGIN OPENSSH PRIVATE KEY") {
		t.Fatalf("PEM block leaked: %q", out)
	}
	if !strings.Contains(out, "trailing") {
		t.Fatalf("non-secret text lost: %q", out)
	}
}

func TestErrorNil(t *testing.T) {
	if Error(nil, "x") != "" {
		t.Fatal("Error(nil) should be empty")
	}
}

func TestErrorRedacts(t *testing.T) {
	const pp = "PASSPHRASE-VALUE-7c2e"
	out := Error(errors.New("parse key: "+pp), pp)
	if strings.Contains(out, pp) {
		t.Fatalf("passphrase leaked: %q", out)
	}
}
