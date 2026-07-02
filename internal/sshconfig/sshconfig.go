// Package sshconfig parses a subset of the OpenSSH client configuration file
// (~/.ssh/config) into simple host entries suitable for import.
//
// Only the basic connection directives are understood: Host, HostName, User,
// Port and IdentityFile. Complex directives (Match, Include, ProxyJump,
// ProxyCommand, LocalForward, RemoteForward, DynamicForward) are intentionally
// NOT interpreted; when found inside a host block they are recorded as a
// warning on that entry so the UI can tell the user they were skipped.
//
// Parse is pure (no filesystem access). Tilde (~) expansion and IdentityFile
// existence checks are the caller's responsibility via ExpandUser.
package sshconfig

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Entry is one importable host derived from a Host block.
type Entry struct {
	Alias        string   `json:"alias"`        // the Host pattern used (first concrete token)
	HostName     string   `json:"hostName"`     // HostName, or Alias if omitted
	User         string   `json:"user"`         // User, may be empty
	Port         int      `json:"port"`         // Port, 0 if unspecified
	IdentityFile string   `json:"identityFile"` // raw path as written (may contain ~)
	Warnings     []string `json:"warnings"`     // skipped directives, defaults applied, etc.
}

// unsupported lists directives we recognize but do not implement in v0.4.0.
var unsupported = map[string]bool{
	"match":          true,
	"include":        true,
	"proxyjump":      true,
	"proxycommand":   true,
	"localforward":   true,
	"remoteforward":  true,
	"dynamicforward": true,
}

func hasWildcard(s string) bool { return strings.ContainsAny(s, "*?!") }

// splitKeyValue splits an OpenSSH config line into its keyword and argument.
// Keyword and argument may be separated by whitespace and/or a single '='.
func splitKeyValue(line string) (string, string) {
	i := strings.IndexAny(line, " \t=")
	if i < 0 {
		return line, ""
	}
	key := line[:i]
	rest := strings.TrimLeft(line[i:], " \t=")
	return key, rest
}

// unquote strips a single pair of surrounding double quotes, if present.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// Parse reads an OpenSSH config and returns one Entry per host block that has
// at least one concrete (non-wildcard) alias. Global blocks such as `Host *`
// are skipped. Parse never touches the filesystem.
func Parse(r io.Reader) ([]Entry, error) {
	var (
		out     []Entry
		cur     *Entry
		curSkip bool // current block is wildcard-only -> ignore its directives
	)

	flush := func() {
		if cur != nil && !curSkip {
			if cur.HostName == "" {
				cur.HostName = cur.Alias
				cur.Warnings = append(cur.Warnings, "缺少 HostName，已使用别名作为地址")
			}
			out = append(out, *cur)
		}
		cur = nil
		curSkip = false
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val := splitKeyValue(line)
		lkey := strings.ToLower(key)

		switch lkey {
		case "host":
			flush()
			patterns := strings.Fields(val)
			alias := ""
			for _, p := range patterns {
				if !hasWildcard(p) {
					alias = p
					break
				}
			}
			if alias == "" {
				// Wildcard-only block (e.g. `Host *`): track it but drop its
				// directives so they don't leak onto the next real block.
				curSkip = true
				cur = &Entry{}
				continue
			}
			cur = &Entry{Alias: alias}
			curSkip = false
		case "hostname":
			if cur != nil && !curSkip {
				cur.HostName = unquote(val)
			}
		case "user":
			if cur != nil && !curSkip {
				cur.User = unquote(val)
			}
		case "port":
			if cur != nil && !curSkip {
				if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					cur.Port = n
				}
			}
		case "identityfile":
			if cur != nil && !curSkip && cur.IdentityFile == "" {
				cur.IdentityFile = unquote(val)
			}
		default:
			if cur != nil && !curSkip && unsupported[lkey] {
				cur.Warnings = append(cur.Warnings, "已跳过不支持的配置项: "+key)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	flush()
	return out, nil
}

// DefaultPath returns the conventional ~/.ssh/config location, or "" if the
// home directory cannot be determined.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

// ExpandUser expands a leading ~ (or ~/) to the user's home directory. Other
// paths are returned unchanged. A best-effort operation: if the home directory
// is unknown, the original path is returned.
func ExpandUser(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if len(path) >= 2 && (path[1] == '/' || path[1] == '\\') {
		return filepath.Join(home, path[2:])
	}
	return path
}
