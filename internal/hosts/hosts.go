// Package hosts manages saved SSH connection profiles (hosts).
// Passwords are encrypted at rest via internal/cryptox.
package hosts

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/leungbzai-png/ssh-terminal/internal/cryptox"
	"github.com/leungbzai-png/ssh-terminal/internal/portable"
)

const file = "hosts.json"

// Host is the user-facing profile.
// Password and Passphrase are plaintext on the wire (Go<->JS) but encrypted on disk.
type Host struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Address    string `json:"address"` // host or host:port
	Port       int    `json:"port"`
	User       string `json:"user"`
	AuthType   string `json:"authType"` // "password" | "key" | "managedKey"
	Password   string `json:"password,omitempty"`
	KeyPath    string `json:"keyPath,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	// ManagedKeyID references an internal/keymgr key when AuthType == "managedKey".
	ManagedKeyID string `json:"managedKeyId,omitempty"`
	Group        string `json:"group,omitempty"`
	Note         string `json:"note,omitempty"`
	UpdatedAt    int64  `json:"updatedAt"`
}

// storedHost mirrors Host but holds encrypted secrets.
type storedHost struct {
	Host
	EncPassword   string `json:"encPassword,omitempty"`
	EncPassphrase string `json:"encPassphrase,omitempty"`
}

var (
	mu    sync.RWMutex
	cache map[string]storedHost
)

func path() string { return portable.DataPath(file) }

func ensureLoaded() error {
	mu.Lock()
	defer mu.Unlock()
	if cache != nil {
		return nil
	}
	cache = map[string]storedHost{}
	data, err := os.ReadFile(path())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var list []storedHost
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	for _, h := range list {
		cache[h.ID] = h
	}
	return nil
}

func persistLocked() error {
	list := make([]storedHost, 0, len(cache))
	for _, h := range cache {
		// Never write plaintext secrets to disk.
		h.Password = ""
		h.Passphrase = ""
		list = append(list, h)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].UpdatedAt > list[j].UpdatedAt })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := path() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path())
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// List returns hosts WITHOUT decrypted secrets (UI never needs them).
func List() ([]Host, error) {
	if err := ensureLoaded(); err != nil {
		return nil, err
	}
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Host, 0, len(cache))
	for _, s := range cache {
		h := s.Host
		h.Password = ""
		h.Passphrase = ""
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Get returns a host with decrypted secrets (for internal SSH connect use only).
func Get(id string) (Host, error) {
	if err := ensureLoaded(); err != nil {
		return Host{}, err
	}
	mu.RLock()
	s, ok := cache[id]
	mu.RUnlock()
	if !ok {
		return Host{}, errors.New("host not found")
	}
	h := s.Host
	if s.EncPassword != "" {
		if pw, err := cryptox.Decrypt(s.EncPassword); err == nil {
			h.Password = pw
		}
	}
	if s.EncPassphrase != "" {
		if pp, err := cryptox.Decrypt(s.EncPassphrase); err == nil {
			h.Passphrase = pp
		}
	}
	return h, nil
}

// Upsert creates or updates a host. ID is assigned if empty.
func Upsert(h Host) (Host, error) {
	if err := ensureLoaded(); err != nil {
		return Host{}, err
	}
	if h.Port == 0 {
		h.Port = 22
	}
	if h.Name == "" {
		h.Name = h.Address
	}
	mu.Lock()
	defer mu.Unlock()
	if h.ID == "" {
		h.ID = newID()
	}
	h.UpdatedAt = time.Now().Unix()
	s := storedHost{Host: h}
	if h.Password != "" {
		enc, err := cryptox.Encrypt(h.Password)
		if err != nil {
			return Host{}, err
		}
		s.EncPassword = enc
	} else if existing, ok := cache[h.ID]; ok {
		// Preserve existing encrypted password if user didn't retype it.
		s.EncPassword = existing.EncPassword
	}
	if h.Passphrase != "" {
		enc, err := cryptox.Encrypt(h.Passphrase)
		if err != nil {
			return Host{}, err
		}
		s.EncPassphrase = enc
	} else if existing, ok := cache[h.ID]; ok {
		s.EncPassphrase = existing.EncPassphrase
	}
	cache[h.ID] = s
	if err := persistLocked(); err != nil {
		return Host{}, err
	}
	out := h
	out.Password = ""
	out.Passphrase = ""
	return out, nil
}

func Delete(id string) error {
	if err := ensureLoaded(); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	if _, ok := cache[id]; !ok {
		return errors.New("host not found")
	}
	delete(cache, id)
	return persistLocked()
}
