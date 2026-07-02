package keymgr

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leungbzai-png/ssh-terminal/internal/cryptox"
)

func resetCache() {
	mu.Lock()
	cache = map[string]storedKey{}
	mu.Unlock()
}

// genPEM produces an OpenSSH private-key PEM, optionally passphrase-protected.
func genPEM(t *testing.T, passphrase string) []byte {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	pemBytes, err := marshalPrivateToPEM(priv, "test@ssh-terminal", passphrase)
	if err != nil {
		t.Fatalf("marshalPrivateToPEM: %v", err)
	}
	return pemBytes
}

// scanKeysDir asserts that no file under data/keys contains the given forbidden
// substrings. The encrypted key (.key.enc) is base64 AES ciphertext, so private
// key material never appears; passphrases are never persisted at all.
func scanKeysDir(t *testing.T, forbidden ...string) {
	t.Helper()
	entries, err := os.ReadDir(keysDir())
	if err != nil {
		t.Fatalf("read keys dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// A plaintext private-key filename must never exist under data/keys.
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".pem") || name == "id_rsa" || name == "id_ed25519" ||
			(strings.HasSuffix(name, ".key") && !strings.HasSuffix(name, ".key.enc")) {
			t.Fatalf("forbidden plaintext key file present: %s", e.Name())
		}
		data, err := os.ReadFile(filepath.Join(keysDir(), e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		s := string(data)
		for _, bad := range forbidden {
			if strings.Contains(s, bad) {
				t.Fatalf("file %s contains forbidden content %q", e.Name(), bad)
			}
		}
	}
}

func TestImportFromFileEncryptsNoPlaintext(t *testing.T) {
	resetCache()
	pemBytes := genPEM(t, "")
	src := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(src, pemBytes, 0o600); err != nil {
		t.Fatalf("write temp key: %v", err)
	}

	k, err := ImportFromFile("imported", "note", src, "")
	if err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}
	if k.HasPassword {
		t.Error("expected HasPassword=false for an unprotected key")
	}
	if k.Fingerprint == "" || k.PublicKey == "" {
		t.Error("expected fingerprint and public key to be derived")
	}
	if k.Type != KeyTypeEd25519 {
		t.Errorf("expected ed25519, got %q", k.Type)
	}

	// No plaintext private-key material anywhere under data/keys.
	scanKeysDir(t, "PRIVATE KEY", "BEGIN OPENSSH PRIVATE KEY", "BEGIN RSA PRIVATE KEY")

	// The encrypted file must round-trip back to the original bytes.
	mu.RLock()
	s := cache[k.ID]
	mu.RUnlock()
	enc, err := os.ReadFile(s.EncryptedPath)
	if err != nil {
		t.Fatalf("read enc: %v", err)
	}
	if !strings.HasSuffix(s.EncryptedPath, ".key.enc") {
		t.Fatalf("encrypted key must be stored as .key.enc, got %s", s.EncryptedPath)
	}
	dec, err := cryptox.Decrypt(string(enc))
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if dec != string(pemBytes) {
		t.Fatal("decrypted key does not match original")
	}

	// The imported key must be usable for auth.
	if _, err := LoadSigner(k.ID, ""); err != nil {
		t.Fatalf("LoadSigner: %v", err)
	}
}

func TestImportPassphraseProtected(t *testing.T) {
	resetCache()
	const pass = "SENTINEL-PASS-7788aa"
	pemBytes := genPEM(t, pass)
	src := filepath.Join(t.TempDir(), "protected_key")
	if err := os.WriteFile(src, pemBytes, 0o600); err != nil {
		t.Fatalf("write temp key: %v", err)
	}

	// Without the passphrase the import must fail (not silently succeed).
	if _, err := ImportFromFile("p1", "", src, ""); err == nil {
		t.Fatal("expected error importing protected key without passphrase")
	}

	k, err := ImportFromFile("p2", "", src, pass)
	if err != nil {
		t.Fatalf("ImportFromFile with passphrase: %v", err)
	}
	if !k.HasPassword {
		t.Error("expected HasPassword=true for a protected key")
	}

	// The passphrase must never be persisted anywhere under data/keys, and no
	// plaintext key material either.
	scanKeysDir(t, pass, "PRIVATE KEY", "BEGIN OPENSSH PRIVATE KEY")

	// Loading requires the passphrase.
	if _, err := LoadSigner(k.ID, pass); err != nil {
		t.Fatalf("LoadSigner with passphrase: %v", err)
	}
}
