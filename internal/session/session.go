// Package session persists the set of open saved-host tabs so they can be
// restored on the next launch.
//
// Security: it stores ONLY non-secret tab intent — a saved host's id and its
// display name. It never stores passwords, passphrases, private keys, Quick
// Connect temporary secrets, terminal buffers, or SFTP state. The authoritative
// host data continues to live (encrypted) in hosts.json.
package session

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/leungbzai-png/ssh-terminal/internal/portable"
)

const file = "session.json"

// OpenTab is a restorable reference to a saved host.
type OpenTab struct {
	HostID   string `json:"hostId"`
	HostName string `json:"hostName"`
}

type state struct {
	OpenTabs []OpenTab `json:"openTabs"`
}

var mu sync.Mutex

func path() string { return portable.DataPath(file) }

// sanitize keeps only non-secret fields and drops entries without a host id
// (e.g. Quick Connect tabs, which must never be persisted).
func sanitize(tabs []OpenTab) []OpenTab {
	out := make([]OpenTab, 0, len(tabs))
	for _, t := range tabs {
		if t.HostID == "" {
			continue
		}
		out = append(out, OpenTab{HostID: t.HostID, HostName: t.HostName})
	}
	return out
}

// Load returns the persisted open tabs (empty slice when none or on any error).
func Load() []OpenTab {
	mu.Lock()
	defer mu.Unlock()
	data, err := os.ReadFile(path())
	if err != nil {
		return []OpenTab{}
	}
	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return []OpenTab{}
	}
	return sanitize(s.OpenTabs)
}

// Save persists the given open tabs, writing only non-secret fields.
func Save(tabs []OpenTab) error {
	mu.Lock()
	defer mu.Unlock()
	data, err := json.MarshalIndent(state{OpenTabs: sanitize(tabs)}, "", "  ")
	if err != nil {
		return err
	}
	tmp := path() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path())
}
