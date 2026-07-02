# Product Roadmap — SSH Terminal

Last updated: 2026-07-02 (v0.4.0 in progress — Part 1: Connection UX)

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
| 1 | **Custom in-app dialogs** (replace SFTP `prompt`/`confirm`) | 3 | 1 | 4.5 | v0.3.0 ✅ |
| 2 | **Import hosts from `~/.ssh/config`** | 5 | 2 | 4.0 | v0.4.0 |
| 3 | **SFTP recursive delete** | 4 | 2 | 3.0 | v0.3.0 ✅ |
| 4 | **Configurable connection timeout** (currently hardcoded 15s) | 3 | 1 | 2.5 | v0.3.0 ✅ |
| 5 | **Quick connect** (connect without saving host) | 4 | 2 | 3.0 | v0.4.0 |
| 6 | **CI / GitHub Actions** (`go vet` + `wails build` on push) | 4 | 1 | 3.5 | v0.3.0 ✅ |
| 7 | **Host export / import** (backup host list) | 3 | 2 | 2.0 | v0.5.0 |
| 8 | **ProxyJump / bastion host** (`-J` style) | 5 | 3 | 3.5 | v0.8.0 |
| 9 | **Local port forwarding** (`-L` style) | 4 | 3 | 2.5 | v0.8.0 |
| 10 | **SSH agent forwarding** | 3 | 2 | 2.0 | v0.8.0 |
| 11 | **Host groups / folders** in sidebar | 3 | 2 | 2.0 | v0.5.0 |
| 12 | **Session keep-alive** (ServerAliveInterval) | 4 | 1 | 3.5 | v0.4.0 |
| 13 | **Unit tests** for `cryptox`, `config`, `keymgr` | 3 | 2 | 2.0 | v0.9.0 |
| 14 | **SFTP file preview** (text/image) | 3 | 3 | 1.5 | v0.7.0 |
| 15 | **Terminal color scheme picker** | 2 | 2 | 1.0 | v0.6.0 |
| 16 | **SFTP two-pane view** (local ↔ remote) | 4 | 4 | 2.0 | v0.7.0 |
| 17 | **Session logging to file** | 3 | 2 | 2.0 | v0.7.0 |
| 18 | **macOS build support** (Wails supports it) | 4 | 4 | 2.0 | post-v1.0.0 |
| 19 | **Keyboard shortcut help / customization** | 2 | 3 | 0.5 | v0.6.0 |
| 20 | **Plugin / extension API** | 3 | 5 | 0.5 | out of scope (non-goal) |

---

## Version Roadmap

### v0.3.0 — Usability & Polish ✅ Released 2026-06-10
**Theme:** Fix rough edges, common workflow improvements

- [x] Replace SFTP `prompt()`/`confirm()` with custom in-app dialogs (`ConfirmDialog` + `InputDialog`)
- [x] SFTP recursive directory delete (with root-path safety guard)
- [x] Configurable connection timeout (global setting, 5–120 s, default 15 s)
- [x] GitHub Actions CI: `go vet` + build on Windows; frontend type-check + build on Ubuntu
- [ ] Import hosts from `~/.ssh/config` — deferred to v0.4.0
- [ ] Quick connect (temporary session without saving host) — deferred to v0.4.0
- [ ] Host export/import (JSON backup) — deferred to v0.4.0
- [ ] Unify UI language — deferred to v0.5.0

**Breaking changes:** None  
**Actual effort:** 1 session

---

## Path to v1.0.0 — Three Parts

The road from v0.3.0 to a stable v1.0.0 is split into three parts. Guiding principles:
**stability first, small iterations, 1–3 core features per version, no large refactors,
no scope creep beyond a lightweight SSH + SFTP client.** No cloud sync, telemetry,
account system, or plugin system will be added.

| Part | Versions | Theme |
|------|----------|-------|
| Part 1 | v0.4.0 | Connection UX |
| Part 2 | v0.5.0 | Host Management + Secure Storage |
| Part 3 | v0.6.0 – v0.9.x → v1.0.0 | Terminal/SFTP Polish + Production Readiness |

---

## Part 1 — v0.4.0 — Connection UX
**Theme:** Make getting connected faster and more resilient. ← **current work**

- [ ] **SSH KeepAlive** — `keepAliveEnabled` + `keepAliveIntervalSec` settings (default on, 30 s); sends `keepalive@openssh.com` after the session is established; goroutine exits cleanly on close
- [ ] **Quick Connect** — connect without saving a host; temporary password/passphrase live in memory only, never written to `hosts.json`; optional "Remember this host" reuses the existing encrypted storage path
- [ ] **Import `~/.ssh/config`** — parse basic OpenSSH config (`Host`, `HostName`, `User`, `Port`, `IdentityFile`); preview before import; skip duplicates; skip/flag complex directives (`Host *`, `Match`, `Include`, `ProxyJump`, forwards); `~` expansion; imported `IdentityFile` is *referenced*, never copied as plaintext into `data/`

**Explicitly NOT in v0.4.0:** ProxyJump, LocalForward, RemoteForward, dynamic SOCKS,
agent forwarding, host groups, password-storage refactor, SFTP two-pane, plugin system.

**Breaking changes:** None (settings additions are backward-compatible; `Manager.Open` gains a `keepAliveSec` parameter — internal only)

---

## Part 2 — v0.5.0 — Host Management + Secure Storage ✅ Released 2026-07-02
**Theme:** Organize many hosts and harden secret handling.

- [x] Host groups / folders in sidebar (Ungrouped virtual group; group field in host dialog)
- [x] Host search (alias/hostname/username/group, case-insensitive; hides empty groups)
- [x] Safe host export / import (JSON backup; **never** exports password / passphrase / private key by default; duplicate-safe import with preview)
- [x] Encrypted private-key import (import an external key → encrypt to `.key.enc`; no plaintext key on disk; passphrase never persisted)
- [x] Security-policy enforcement (automated tests assert no plaintext secrets are ever persisted / exported)
- [x] No plaintext secrets on disk (whitelist export struct + sentinel/PEM-marker scan tests)

**Status:** Released. Manual QA A–I passed; tag `v0.5.0` created and GitHub Release published.

**Breaking changes:** None. `hosts.json` schema is unchanged (`group` field already existed); export/import is additive.

---

## Part 3 — v0.6.0 → v1.0.0 — Terminal/SFTP Polish + Production Readiness
**Theme:** Refine the day-to-day experience and get to a stable release.

### v0.6.0 — Terminal UX
- [ ] Terminal search improvements
- [ ] Font settings (family/size/ligatures) polish
- [ ] Tab restore (reopen last session set)
- [ ] Keyboard shortcut help panel (Ctrl+? reference)

### v0.7.0 — SFTP UX
- [ ] Transfer progress refinements
- [ ] Drag-upload polish
- [ ] Remote directory bookmarks
- [ ] Optional file preview (text/image)

### v0.8.0 — Advanced SSH
- [ ] ProxyJump / bastion host
- [ ] Local / remote port forwarding
- [ ] Dynamic SOCKS proxy
- [ ] Auto reconnect

### v0.9.0 — Hardening
- [ ] Unit tests: `cryptox`, `config`, `keymgr` (and `sshconfig` parser)
- [ ] CI coverage expansion
- [ ] Documentation + QA pass
- [ ] Performance profiling (startup, idle memory, large output)

### v1.0.0 — Stable
- [ ] Stable tag only — **no new major feature**
- [ ] At least one release cycle with no Critical/High bugs
- [ ] CHANGELOG maintained through the full 0.4–0.9 cycle

**Estimated timeline:** paced by stability, not calendar.

---

## Non-Goals

These will likely never be in scope for this project:

- **Web/browser version** — Wails is desktop-only by design
- **Android/iOS** — wrong platform
- **Serial port / telnet** — scope creep; use Tabby for those
- **Built-in terminal multiplexer** (tmux/screen replacement) — use the real tools
- **Cloud sync of credentials** — incompatible with the security model (local-only encryption)
- **Telemetry / analytics** — zero network calls beyond user-initiated SSH/SFTP is a core promise
- **Account system / login** — the app is local and portable; no identity layer
- **Plugin / extension system** — keeps the surface small and the build reproducible
