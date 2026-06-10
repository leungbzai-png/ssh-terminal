// Package cryptox provides authenticated symmetric encryption for stored
// credentials. The key is generated on first run and persisted next to the
// executable. This protects against casual disk inspection; for stronger
// protection prefer SSH key-based authentication and never store passwords.
package cryptox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/leungbzai-png/ssh-terminal/internal/portable"
)

const keyFile = "secret.key"
const keyLen = 32 // AES-256

var (
	mu  sync.Mutex
	key []byte
)

func loadOrCreateKey() ([]byte, error) {
	mu.Lock()
	defer mu.Unlock()
	if key != nil {
		return key, nil
	}
	path := portable.DataPath(keyFile)
	if data, err := os.ReadFile(path); err == nil && len(data) == keyLen {
		key = data
		return key, nil
	}
	k := make([]byte, keyLen)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	// 0600: owner read/write only.
	if err := os.WriteFile(path, k, 0o600); err != nil {
		return nil, err
	}
	key = k
	return key, nil
}

// Encrypt returns base64(nonce || ciphertext || tag). Empty input -> empty output.
func Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	k, err := loadOrCreateKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt reverses Encrypt. Empty input -> empty output.
func Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	k, err := loadOrCreateKey()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
