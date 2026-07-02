package keymgr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/leungbzai-png/ssh-terminal/internal/cryptox"
)

// ImportFromFile imports an existing private key from an external file into the
// managed key store.
//
// Security model (must not regress):
//   - The key file is read on the Go side only; its plaintext bytes never cross
//     the Wails bridge and are never logged.
//   - The ORIGINAL key bytes are stored, encrypted with cryptox, to
//     data/keys/<id>.key.enc. No plaintext private-key file is ever written
//     under data/.
//   - A passphrase, if supplied, is used only transiently to validate a
//     passphrase-protected key. It is NEVER persisted and NEVER used to strip
//     the passphrase (the encrypted-at-rest key keeps its original protection).
//   - Only the public key, fingerprint, and metadata are recorded in index.json.
func ImportFromFile(name, comment, path, passphrase string) (Key, error) {
	if err := ensureLoaded(); err != nil {
		return Key{}, err
	}
	if name == "" {
		return Key{}, errors.New("name required")
	}
	if path == "" {
		return Key{}, errors.New("key path required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Key{}, err
	}

	// Validate and derive the public key. Try without a passphrase first; if the
	// key is passphrase-protected, x/crypto returns *ssh.PassphraseMissingError.
	var signer ssh.Signer
	hasPassword := false
	signer, err = ssh.ParsePrivateKey(data)
	if err != nil {
		var missing *ssh.PassphraseMissingError
		if errors.As(err, &missing) {
			if passphrase == "" {
				return Key{}, errors.New("私钥受口令保护，请提供口令")
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
			if err != nil {
				return Key{}, errors.New("口令错误或私钥无效")
			}
			hasPassword = true
		} else {
			return Key{}, errors.New("不支持或无效的私钥文件")
		}
	}

	pubLine := signer.PublicKey()
	id := newID()

	// Encrypt the ORIGINAL key bytes at rest. Never write plaintext.
	encPath := filepath.Join(keysDir(), id+".key.enc")
	encB64, err := cryptox.Encrypt(string(data))
	if err != nil {
		return Key{}, err
	}
	if err := os.WriteFile(encPath, []byte(encB64), 0o600); err != nil {
		return Key{}, err
	}

	authorizedLine := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pubLine)))
	if comment != "" && !strings.Contains(authorizedLine, comment) {
		authorizedLine += " " + comment
	}
	_ = os.WriteFile(filepath.Join(keysDir(), id+".pub"), []byte(authorizedLine+"\n"), 0o644)

	k := Key{
		ID:          id,
		Name:        name,
		Type:        keyTypeFromPub(pubLine.Type()),
		Comment:     comment,
		Fingerprint: ssh.FingerprintSHA256(pubLine),
		PublicKey:   authorizedLine,
		HasPassword: hasPassword,
		CreatedAt:   time.Now().Unix(),
	}
	mu.Lock()
	cache[id] = storedKey{Key: k, EncryptedPath: encPath}
	err = persistLocked()
	mu.Unlock()
	if err != nil {
		return Key{}, err
	}
	return k, nil
}

// keyTypeFromPub maps an SSH public-key algorithm name to a KeyType label for
// display. Unknown algorithms are recorded verbatim (metadata only).
func keyTypeFromPub(algo string) KeyType {
	switch {
	case strings.Contains(algo, "ed25519"):
		return KeyTypeEd25519
	case strings.Contains(algo, "rsa"):
		return KeyTypeRSA
	case strings.Contains(algo, "ecdsa"):
		return KeyType("ecdsa")
	case strings.Contains(algo, "dss"), strings.Contains(algo, "dsa"):
		return KeyType("dsa")
	default:
		return KeyType(fmt.Sprintf("%.32s", algo))
	}
}
