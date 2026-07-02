// Package bookmarks stores per-host SFTP remote-path bookmarks.
//
// Security: a bookmark holds only non-secret data — a display name, a remote
// path, and the owning saved-host id. It never contains a password, passphrase,
// or any key material. Bookmarks live in their own file (data/bookmarks.json),
// separate from host secure storage, and are NOT part of the safe host export.
package bookmarks

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leungbzai-png/ssh-terminal/internal/portable"
)

const file = "bookmarks.json"

// Bookmark is a saved remote path for a specific host.
type Bookmark struct {
	ID        string `json:"id"`
	HostID    string `json:"hostId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	CreatedAt int64  `json:"createdAt"`
}

var (
	mu     sync.Mutex
	cache  []Bookmark
	loaded bool
)

func filePath() string { return portable.DataPath(file) }

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func ensureLoadedLocked() error {
	if loaded {
		return nil
	}
	cache = []Bookmark{}
	data, err := os.ReadFile(filePath())
	if os.IsNotExist(err) {
		loaded = true
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		// Corrupt file: start empty rather than crash.
		cache = []Bookmark{}
	}
	loaded = true
	return nil
}

func persistLocked() error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	tmp := filePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filePath())
}

// List returns the bookmarks for a host, sorted by name. Empty hostID yields no
// results (bookmarks are always host-scoped).
func List(hostID string) ([]Bookmark, error) {
	mu.Lock()
	defer mu.Unlock()
	if err := ensureLoadedLocked(); err != nil {
		return nil, err
	}
	out := make([]Bookmark, 0)
	for _, b := range cache {
		if hostID != "" && b.HostID == hostID {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

// Add creates a bookmark. name defaults to the path when empty. Adding the same
// host+path again is idempotent (returns the existing bookmark).
func Add(hostID, name, path string) (Bookmark, error) {
	path = strings.TrimSpace(path)
	if hostID == "" {
		return Bookmark{}, errors.New("host required for bookmark")
	}
	if path == "" {
		return Bookmark{}, errors.New("path required for bookmark")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = path
	}
	mu.Lock()
	defer mu.Unlock()
	if err := ensureLoadedLocked(); err != nil {
		return Bookmark{}, err
	}
	for _, b := range cache {
		if b.HostID == hostID && b.Path == path {
			return b, nil // idempotent
		}
	}
	b := Bookmark{
		ID:        newID(),
		HostID:    hostID,
		Name:      name,
		Path:      path,
		CreatedAt: time.Now().Unix(),
	}
	cache = append(cache, b)
	if err := persistLocked(); err != nil {
		return Bookmark{}, err
	}
	return b, nil
}

// Delete removes a bookmark by id (no error if it does not exist).
func Delete(id string) error {
	mu.Lock()
	defer mu.Unlock()
	if err := ensureLoadedLocked(); err != nil {
		return err
	}
	next := cache[:0]
	for _, b := range cache {
		if b.ID != id {
			next = append(next, b)
		}
	}
	cache = next
	return persistLocked()
}
