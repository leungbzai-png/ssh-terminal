// Package config stores user preferences (theme, font, etc.) in JSON.
package config

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/leungbzai-png/ssh-terminal/internal/portable"
)

const file = "settings.json"

type Settings struct {
	Theme       string `json:"theme"`       // "light" | "dark" | "system"
	FontFamily  string `json:"fontFamily"`  // monospace stack
	FontSize    int    `json:"fontSize"`    // px
	CursorStyle string `json:"cursorStyle"` // "block" | "bar" | "underline"
	CursorBlink bool   `json:"cursorBlink"`
	ScrollBack  int    `json:"scrollBack"` // lines
	// ConfirmCloseWithActiveSessions: when true, closing the app while at least
	// one SSH session is open will prompt the user for confirmation.
	ConfirmCloseWithActiveSessions bool `json:"confirmCloseWithActiveSessions"`
	// ShowCommandBar: show the FinalShell-style command input at the bottom of each terminal.
	ShowCommandBar bool `json:"showCommandBar"`
	// ConnectTimeoutSec is the TCP+SSH handshake timeout in seconds.
	ConnectTimeoutSec int `json:"connectTimeoutSec"`
	// KeepAliveEnabled, when true, sends periodic keepalive@openssh.com requests
	// on each live session to keep idle connections and NAT mappings alive.
	KeepAliveEnabled bool `json:"keepAliveEnabled"`
	// KeepAliveIntervalSec is the interval between keepalive requests in seconds.
	KeepAliveIntervalSec int `json:"keepAliveIntervalSec"`
}

func Defaults() Settings {
	return Settings{
		Theme:                          "system",
		FontFamily:                     `"JetBrains Mono","Cascadia Code","Fira Code",Consolas,monospace`,
		FontSize:                       14,
		CursorStyle:                    "bar",
		CursorBlink:                    true,
		ScrollBack:                     5000,
		ConfirmCloseWithActiveSessions: true,
		ShowCommandBar:                 true,
		ConnectTimeoutSec:              15,
		KeepAliveEnabled:               true,
		KeepAliveIntervalSec:           30,
	}
}

var (
	mu      sync.RWMutex
	current Settings
	loaded  bool
)

func path() string { return portable.DataPath(file) }

func Load() Settings {
	mu.Lock()
	defer mu.Unlock()
	if loaded {
		return current
	}
	current = Defaults()
	data, err := os.ReadFile(path())
	if err == nil {
		_ = json.Unmarshal(data, &current)
	}
	loaded = true
	return current
}

func Save(s Settings) error {
	mu.Lock()
	defer mu.Unlock()
	current = s
	loaded = true
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path(), data, 0o644)
}
