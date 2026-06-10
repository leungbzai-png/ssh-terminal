# Product Roadmap — SSH Terminal

Last updated: 2026-06-10 (v0.2.0 released)

---

## Competitive Positioning

### vs. FinalShell

| Dimension | FinalShell | SSH Terminal |
|-----------|-----------|--------------|
| License | Closed source, freemium | MIT open source |
| Privacy | Telemetry / cloud features | Zero network calls beyond SSH |
| Size | ~80 MB installer | ~11 MB portable exe |
| File manager | Built-in with preview | SFTP panel (basic) |
| SSH features | Very complete (tunnel, jump) | Single-hop only |
| Terminal quality | Good | Good (xterm.js) |
| UI language | Chinese-primary | Chinese (mixed some English) |
| Monitoring | CPU/mem graphs over SSH | Not available |
| Platform | Windows, macOS | Windows only |
| **Key gap vs us** | ← We lack jump host, tunneling, monitoring |
| **Key advantage** | → We are open source, portable, no bloat |

### vs. Xshell (NetSarang)

| Dimension | Xshell | SSH Terminal |
|-----------|--------|--------------|
| License | Commercial (free for personal) | MIT open source |
| SSH features | Industry-leading (agent, jump, tunnel, port-fwd) | Basic |
| Scripting | Full scripting support | None |
| UI | Dense, professional | Clean, minimal |
| Portability | Registry-based | Fully portable |
| Performance | Native | WebView2 (slight overhead) |
| **Key gap vs us** | ← We lack scripting, tunneling, agent forwarding |
| **Key advantage** | → We are free for all use, portable, open source |

### vs. Tabby (Electron)

| Dimension | Tabby | SSH Terminal |
|-----------|-------|--------------|
| License | MIT open source | MIT open source |
| Platform | Windows, macOS, Linux | Windows only |
| Size | ~120 MB | ~11 MB |
| Plugin system | Rich plugin ecosystem | No plugins |
| SSH features | Good, with tunneling | Basic |
| Serial port | Yes | No |
| Startup speed | Slow (Electron) | Fast (WebView2) |
| Memory usage | ~250 MB idle | ~80 MB idle (estimated) |
| Community | Large | Early stage |
| **Key gap vs us** | ← We lack cross-platform, plugins, tunneling |
| **Key advantage** | → We are 10× lighter, faster to start, simpler |

### Strategic Positioning

SSH Terminal occupies a unique space: **lighter than Tabby, more open than Xshell, cleaner than FinalShell**. The target user is a developer or sysadmin on Windows who wants a fast, portable, no-registration SSH tool with SFTP built in. The right growth path is deepening core SSH/SFTP features before adding complexity.

---

## Top 20 Features to Build

Scoring: **Benefit** 1–5 (user value), **Difficulty** 1–5 (engineering effort), **Priority** = Benefit − Difficulty/2 (higher = do sooner)

| # | Feature | Benefit | Difficulty | Priority | Version |
|---|---------|---------|------------|----------|---------|
| 1 | **Custom in-app dialogs** (replace SFTP `prompt`/`confirm`) | 3 | 1 | 4.5 | v0.3.0 |
| 2 | **Import hosts from `~/.ssh/config`** | 5 | 2 | 4.0 | v0.3.0 |
| 3 | **SFTP recursive delete** | 4 | 2 | 3.0 | v0.3.0 |
| 4 | **Configurable connection timeout** (currently hardcoded 15s) | 3 | 1 | 2.5 | v0.3.0 |
| 5 | **Quick connect** (connect without saving host) | 4 | 2 | 3.0 | v0.3.0 |
| 6 | **CI / GitHub Actions** (`go vet` + `wails build` on push) | 4 | 1 | 3.5 | v0.3.0 |
| 7 | **Host export / import** (backup host list) | 3 | 2 | 2.0 | v0.3.0 |
| 8 | **ProxyJump / bastion host** (`-J` style) | 5 | 3 | 3.5 | v0.4.0 |
| 9 | **Local port forwarding** (`-L` style) | 4 | 3 | 2.5 | v0.4.0 |
| 10 | **SSH agent forwarding** | 3 | 2 | 2.0 | v0.4.0 |
| 11 | **Host groups / folders** in sidebar | 3 | 2 | 2.0 | v0.4.0 |
| 12 | **Session keep-alive** (ServerAliveInterval) | 4 | 1 | 3.5 | v0.4.0 |
| 13 | **Unit tests** for `cryptox`, `config`, `keymgr` | 3 | 2 | 2.0 | v0.4.0 |
| 14 | **SFTP file preview** (text/image) | 3 | 3 | 1.5 | v0.5.0 |
| 15 | **Terminal color scheme picker** | 2 | 2 | 1.0 | v0.5.0 |
| 16 | **SFTP two-pane view** (local ↔ remote) | 4 | 4 | 2.0 | v0.5.0 |
| 17 | **Session logging to file** | 3 | 2 | 2.0 | v0.5.0 |
| 18 | **macOS build support** (Wails supports it) | 4 | 4 | 2.0 | v1.0.0 |
| 19 | **Keyboard shortcut customization** | 2 | 3 | 0.5 | v1.0.0 |
| 20 | **Plugin / extension API** | 3 | 5 | 0.5 | v1.0.0+ |

---

## Version Roadmap

### v0.3.0 — Usability & Polish
**Theme:** Fix rough edges, common workflow improvements

- [ ] Replace SFTP `prompt()`/`confirm()` with custom in-app dialogs
- [ ] SFTP recursive directory delete
- [ ] Import hosts from `~/.ssh/config`
- [ ] Quick connect (temporary session without saving host)
- [ ] Configurable per-host connection timeout
- [ ] Host export/import (JSON backup)
- [ ] GitHub Actions CI: `go vet` + build check on every push
- [ ] Unify UI language (all Chinese, or offer language setting)

**Breaking changes:** None  
**Estimated effort:** 3–4 weeks solo

---

### v0.4.0 — Advanced SSH
**Theme:** Close the gap with Xshell/Tabby on core SSH features

- [ ] ProxyJump / bastion host support
- [ ] Local port forwarding (`-L` style)
- [ ] SSH agent forwarding
- [ ] ServerAliveInterval keep-alive per host
- [ ] Host grouping / folders in sidebar
- [ ] Unit tests for `cryptox`, `portable`, `config`, `keymgr`
- [ ] Keyboard shortcut reference (Ctrl+? help panel)

**Breaking changes:** `hosts.json` may need schema addition for new per-host options  
**Estimated effort:** 6–8 weeks solo

---

### v0.5.0 — File Management
**Theme:** SFTP as a first-class feature

- [ ] SFTP two-pane view (local left, remote right, drag between)
- [ ] SFTP file preview (text files, images)
- [ ] Session/terminal logging to file
- [ ] Terminal color scheme picker (presets + custom)
- [ ] SFTP bookmark/favorites for remote directories
- [ ] Batch download (select multiple, download as zip)

**Breaking changes:** None  
**Estimated effort:** 6–8 weeks solo

---

### v1.0.0 — Stability & Reach
**Theme:** Production-ready for all users

**Prerequisites (all must be complete before tagging v1.0.0):**
- [ ] At least one release cycle with no Critical/High bugs reported
- [ ] Unit test coverage for all crypto and config paths
- [ ] CI pipeline passing on all PRs
- [ ] macOS build explored (Wails supports it; needs code signing)
- [ ] CHANGELOG maintained through at least 3 release cycles
- [ ] Performance profiling (startup time, memory at idle, large terminal output)
- [ ] Accessibility: keyboard-only navigation for dialogs
- [ ] Documentation for users (beyond README)

**Estimated timeline:** 4–6 months from v0.5.0

---

## Non-Goals

These will likely never be in scope for this project:

- **Web/browser version** — Wails is desktop-only by design
- **Android/iOS** — wrong platform
- **Serial port / telnet** — scope creep; use Tabby for those
- **Built-in terminal multiplexer** (tmux/screen replacement) — use the real tools
- **Cloud sync of credentials** — incompatible with the security model (local-only encryption)
