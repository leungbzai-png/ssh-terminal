// Package keymgr manages locally generated SSH keypairs.
//
// Public keys are stored in plain text under data/keys/<id>.pub.
// Private keys are AES-256-GCM encrypted via internal/cryptox and written to
// data/keys/<id>.key.enc, so disk reads alone can't recover them.
// A metadata index lives at data/keys/index.json.
package keymgr

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/leungbzai-png/ssh-terminal/internal/cryptox"
	"github.com/leungbzai-png/ssh-terminal/internal/portable"
)

type KeyType string

const (
	KeyTypeEd25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
)

type Key struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        KeyType `json:"type"`
	Comment     string  `json:"comment"`
	Fingerprint string  `json:"fingerprint"`
	PublicKey   string  `json:"publicKey"`
	HasPassword bool    `json:"hasPassword"`
	CreatedAt   int64   `json:"createdAt"`
}

type storedKey struct {
	Key
	EncryptedPath string `json:"encryptedPath"`
}

const indexFile = "index.json"

var (
	mu    sync.RWMutex
	cache map[string]storedKey
)

func keysDir() string {
	d := filepath.Join(portable.DataDir(), "keys")
	_ = os.MkdirAll(d, 0o700)
	return d
}

func indexPath() string { return filepath.Join(keysDir(), indexFile) }

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func ensureLoaded() error {
	mu.Lock()
	defer mu.Unlock()
	if cache != nil {
		return nil
	}
	cache = map[string]storedKey{}
	data, err := os.ReadFile(indexPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var list []storedKey
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	for _, k := range list {
		cache[k.ID] = k
	}
	return nil
}

func persistLocked() error {
	list := make([]storedKey, 0, len(cache))
	for _, k := range cache {
		list = append(list, k)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt > list[j].CreatedAt })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := indexPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, indexPath())
}

// marshalPrivateToPEM encodes a crypto.PrivateKey to OpenSSH PEM,
// optionally encrypted with a passphrase.
func marshalPrivateToPEM(sk crypto.PrivateKey, comment string, passphrase string) ([]byte, error) {
	var block *pem.Block
	var err error
	if passphrase != "" {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(sk, comment, []byte(passphrase))
	} else {
		block, err = ssh.MarshalPrivateKey(sk, comment)
	}
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(block), nil
}

// Generate creates a new key, writes it to disk, and returns its metadata.
func Generate(name, comment string, keyType KeyType, rsaBits int, passphrase string) (Key, error) {
	if err := ensureLoaded(); err != nil {
		return Key{}, err
	}
	if name == "" {
		return Key{}, errors.New("name required")
	}

	var (
		privSigner crypto.PrivateKey
		pubLine    ssh.PublicKey
		err        error
	)
	switch keyType {
	case KeyTypeEd25519:
		pub, priv, gerr := ed25519.GenerateKey(rand.Reader)
		if gerr != nil {
			return Key{}, gerr
		}
		privSigner = priv
		pubLine, err = ssh.NewPublicKey(pub)
		if err != nil {
			return Key{}, err
		}
	case KeyTypeRSA:
		if rsaBits < 2048 {
			rsaBits = 4096
		}
		k, gerr := rsa.GenerateKey(rand.Reader, rsaBits)
		if gerr != nil {
			return Key{}, gerr
		}
		privSigner = k
		pubLine, err = ssh.NewPublicKey(&k.PublicKey)
		if err != nil {
			return Key{}, err
		}
	default:
		return Key{}, fmt.Errorf("unsupported key type: %s", keyType)
	}

	privPEM, err := marshalPrivateToPEM(privSigner, comment, passphrase)
	if err != nil {
		return Key{}, err
	}

	id := newID()
	encPath := filepath.Join(keysDir(), id+".key.enc")
	encB64, err := cryptox.Encrypt(string(privPEM))
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
		Type:        keyType,
		Comment:     comment,
		Fingerprint: ssh.FingerprintSHA256(pubLine),
		PublicKey:   authorizedLine,
		HasPassword: passphrase != "",
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

func List() ([]Key, error) {
	if err := ensureLoaded(); err != nil {
		return nil, err
	}
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Key, 0, len(cache))
	for _, s := range cache {
		out = append(out, s.Key)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}

func Get(id string) (Key, error) {
	if err := ensureLoaded(); err != nil {
		return Key{}, err
	}
	mu.RLock()
	defer mu.RUnlock()
	s, ok := cache[id]
	if !ok {
		return Key{}, errors.New("key not found")
	}
	return s.Key, nil
}

// LoadSigner returns an ssh.Signer ready for ssh.PublicKeys auth.
func LoadSigner(id, passphrase string) (ssh.Signer, error) {
	if err := ensureLoaded(); err != nil {
		return nil, err
	}
	mu.RLock()
	s, ok := cache[id]
	mu.RUnlock()
	if !ok {
		return nil, errors.New("key not found")
	}
	enc, err := os.ReadFile(s.EncryptedPath)
	if err != nil {
		return nil, err
	}
	priv, err := cryptox.Decrypt(string(enc))
	if err != nil {
		return nil, err
	}
	if s.HasPassword {
		return ssh.ParsePrivateKeyWithPassphrase([]byte(priv), []byte(passphrase))
	}
	return ssh.ParsePrivateKey([]byte(priv))
}

func Delete(id string) error {
	if err := ensureLoaded(); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	s, ok := cache[id]
	if !ok {
		return errors.New("key not found")
	}
	_ = os.Remove(s.EncryptedPath)
	_ = os.Remove(filepath.Join(keysDir(), s.ID+".pub"))
	delete(cache, s.ID)
	return persistLocked()
}

func PublicKey(id string) (string, error) {
	k, err := Get(id)
	if err != nil {
		return "", err
	}
	return k.PublicKey, nil
}
