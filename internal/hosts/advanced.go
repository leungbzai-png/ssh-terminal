package hosts

import (
	"fmt"
	"strings"
)

// Advanced SSH configuration (v0.8.0).
//
// Security: every field in this file is NON-SECRET. It carries only connection
// topology and tunnel metadata. A jump host's credentials are never stored
// here — a password-authenticated bastion MUST reference a saved host by ID
// (ProxyJump.Mode == "savedHost"), whose secrets remain in secure storage and
// are decrypted only at connect time. A "manual" bastion may reference an
// external key file by path only; it can never carry a password/passphrase.
// Because AdvancedSSH holds no secrets, it is safe to include in the plain
// host export.

// ProxyJump mode constants.
const (
	ProxyJumpSavedHost = "savedHost"
	ProxyJumpManual    = "manual"
)

// Bounds shared by tunnel/auto-reconnect validation.
const (
	maxTunnelNameLen        = 64
	defaultLocalBind        = "127.0.0.1"
	defaultReconnectMax     = 3
	defaultReconnectDelay   = 3
	maxReconnectAttempts    = 10
	maxReconnectDelaySecond = 60
)

// AdvancedSSH holds optional Advanced SSH settings for a host.
type AdvancedSSH struct {
	ProxyJump       *ProxyJump       `json:"proxyJump,omitempty"`
	LocalForwards   []Forward        `json:"localForwards,omitempty"`
	RemoteForwards  []Forward        `json:"remoteForwards,omitempty"`
	DynamicForwards []DynamicForward `json:"dynamicForwards,omitempty"`
	AutoReconnect   *AutoReconnect   `json:"autoReconnect,omitempty"`
}

// ProxyJump configures a single bastion / jump host.
type ProxyJump struct {
	// Mode is ProxyJumpSavedHost or ProxyJumpManual.
	Mode       string `json:"mode"`
	JumpHostID string `json:"jumpHostId,omitempty"` // when Mode == savedHost
	Address    string `json:"address,omitempty"`    // when Mode == manual
	Port       int    `json:"port,omitempty"`
	User       string `json:"user,omitempty"`
	KeyPath    string `json:"keyPath,omitempty"` // external key reference only
}

// Forward describes a local or remote port forward.
type Forward struct {
	Name       string `json:"name,omitempty"`
	LocalHost  string `json:"localHost,omitempty"`  // default 127.0.0.1
	LocalPort  int    `json:"localPort,omitempty"`
	RemoteHost string `json:"remoteHost,omitempty"` // for remote forwards, server bind
	RemotePort int    `json:"remotePort,omitempty"`
	Enabled    bool   `json:"enabled"`
}

// DynamicForward describes a local SOCKS5 dynamic forward.
type DynamicForward struct {
	Name      string `json:"name,omitempty"`
	LocalHost string `json:"localHost,omitempty"` // default 127.0.0.1
	LocalPort int    `json:"localPort,omitempty"`
	Enabled   bool   `json:"enabled"`
}

// AutoReconnect controls automatic reconnection after an unexpected drop.
type AutoReconnect struct {
	Enabled      bool `json:"enabled"`
	MaxAttempts  int  `json:"maxAttempts"`  // 0..10
	DelaySeconds int  `json:"delaySeconds"` // 1..60
}

// validatePort returns an error if p is outside the valid TCP port range.
func validatePort(label string, p int) error {
	if p < 1 || p > 65535 {
		return fmt.Errorf("%s 端口必须在 1-65535 之间（当前 %d）", label, p)
	}
	return nil
}

// bindHost returns the effective local bind host, defaulting to 127.0.0.1.
func bindHost(h string) string {
	h = strings.TrimSpace(h)
	if h == "" {
		return defaultLocalBind
	}
	return h
}

// IsWildcardBind reports whether a bind host exposes the tunnel beyond
// localhost (0.0.0.0 / :: / empty-as-all). Used to surface a warning; binding
// is still allowed if the user explicitly opts in.
func IsWildcardBind(h string) bool {
	switch strings.TrimSpace(h) {
	case "0.0.0.0", "::", "*":
		return true
	}
	return false
}

func validateName(name string) error {
	if len(name) > maxTunnelNameLen {
		return fmt.Errorf("名称过长（最多 %d 字符）", maxTunnelNameLen)
	}
	return nil
}

// Normalize fills defaults and validates the Advanced SSH configuration. It is
// safe to call on a nil receiver (returns nil). Only ENABLED forwards are hard-
// validated for ports/hosts, so a half-filled disabled draft never blocks a
// save; every forward still gets its local bind defaulted to 127.0.0.1.
func (a *AdvancedSSH) Normalize() error {
	if a == nil {
		return nil
	}
	if a.ProxyJump != nil {
		if err := a.ProxyJump.normalize(); err != nil {
			return err
		}
	}
	// Track enabled local binds to catch duplicate host:port collisions across
	// local + dynamic forwards (both bind a local listener).
	seenLocal := map[string]string{}
	for i := range a.LocalForwards {
		f := &a.LocalForwards[i]
		f.LocalHost = bindHost(f.LocalHost)
		if err := validateName(f.Name); err != nil {
			return fmt.Errorf("本地转发: %w", err)
		}
		if !f.Enabled {
			continue
		}
		if err := validatePort("本地转发本地", f.LocalPort); err != nil {
			return err
		}
		if err := validatePort("本地转发远程", f.RemotePort); err != nil {
			return err
		}
		if strings.TrimSpace(f.RemoteHost) == "" {
			return fmt.Errorf("本地转发 %q 缺少远程主机", f.Name)
		}
		key := fmt.Sprintf("%s:%d", f.LocalHost, f.LocalPort)
		if prev, ok := seenLocal[key]; ok {
			return fmt.Errorf("本地绑定端口冲突 %s（%q 与 %q）", key, prev, f.Name)
		}
		seenLocal[key] = f.Name
	}
	for i := range a.DynamicForwards {
		f := &a.DynamicForwards[i]
		f.LocalHost = bindHost(f.LocalHost)
		if err := validateName(f.Name); err != nil {
			return fmt.Errorf("动态转发: %w", err)
		}
		if !f.Enabled {
			continue
		}
		if err := validatePort("动态转发本地", f.LocalPort); err != nil {
			return err
		}
		key := fmt.Sprintf("%s:%d", f.LocalHost, f.LocalPort)
		if prev, ok := seenLocal[key]; ok {
			return fmt.Errorf("本地绑定端口冲突 %s（%q 与 %q）", key, prev, f.Name)
		}
		seenLocal[key] = f.Name
	}
	for i := range a.RemoteForwards {
		f := &a.RemoteForwards[i]
		f.LocalHost = bindHost(f.LocalHost)
		if err := validateName(f.Name); err != nil {
			return fmt.Errorf("远程转发: %w", err)
		}
		if !f.Enabled {
			continue
		}
		if err := validatePort("远程转发远程", f.RemotePort); err != nil {
			return err
		}
		if err := validatePort("远程转发本地", f.LocalPort); err != nil {
			return err
		}
	}
	if a.AutoReconnect != nil {
		a.AutoReconnect.normalize()
	}
	return nil
}

func (p *ProxyJump) normalize() error {
	switch p.Mode {
	case ProxyJumpSavedHost:
		if strings.TrimSpace(p.JumpHostID) == "" {
			return fmt.Errorf("跳板机（引用已保存主机）缺少主机选择")
		}
	case ProxyJumpManual:
		if strings.TrimSpace(p.Address) == "" {
			return fmt.Errorf("跳板机（手动）缺少地址")
		}
		if strings.TrimSpace(p.User) == "" {
			return fmt.Errorf("跳板机（手动）缺少用户名")
		}
		if p.Port == 0 {
			p.Port = 22
		}
		if err := validatePort("跳板机", p.Port); err != nil {
			return err
		}
		// Manual mode is key-only by design; there is nowhere to put a password.
	default:
		return fmt.Errorf("未知跳板机模式: %q", p.Mode)
	}
	return nil
}

func (r *AutoReconnect) normalize() {
	if r.MaxAttempts < 0 {
		r.MaxAttempts = 0
	}
	if r.MaxAttempts > maxReconnectAttempts {
		r.MaxAttempts = maxReconnectAttempts
	}
	if r.MaxAttempts == 0 && r.Enabled {
		r.MaxAttempts = defaultReconnectMax
	}
	if r.DelaySeconds < 1 {
		r.DelaySeconds = defaultReconnectDelay
	}
	if r.DelaySeconds > maxReconnectDelaySecond {
		r.DelaySeconds = maxReconnectDelaySecond
	}
}
