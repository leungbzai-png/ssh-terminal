# GitHub Release Reference

This file is a template for creating GitHub Releases and managing the repository description.
Update the "Current Release" section before each new release.

---

## Current Release: v0.2.0

### Release Title
```
SSH Terminal v0.2.0
```

### Release Notes (paste into GitHub Release body)

```markdown
## SSH Terminal v0.2.0

A lightweight, portable SSH terminal for Windows — built with Go + Wails + Vue 3.

### Features in this release

- **Split-pane terminal** — up to 4 side-by-side panes, each with its own tab bar
- **SFTP file browser** — browse, upload, download, rename, mkdir, delete per session
- **Drag-and-drop upload** — drop files/folders from Explorer onto any active session
- **SSH keypair management** — generate Ed25519 / RSA keys, deploy public keys to hosts
- **Command broadcast bar** — send commands to a single tab or all tabs in a pane
- **Encrypted credential storage** — AES-256-GCM; passwords never written to disk in plaintext
- **Strict host key verification** — first-connect fingerprint confirmation, MITM detection
- **Light / dark / system theme** — xterm.js palette synchronized with CSS design tokens
- **Portable** — all data lives next to the exe in `data/`; no registry, no installer

### Requirements

- Windows 10 / 11
- WebView2 Runtime (included in Win11; Win10 prompts on first launch)
- No installation needed — copy the exe anywhere and run

### Download

| File | Description |
|------|-------------|
| `ssh-terminal.exe` | Portable Windows executable (x64) |

### Build from source

```
git clone https://github.com/leungbzai-png/ssh-terminal.git
cd ssh-terminal
build-windows.bat
```

Requires Go 1.22+ and Node.js 18+ on PATH.

### Notes

- This is the first public release. Core SSH and SFTP functionality is stable.
- All user data is stored in `data/` next to the exe. Back up this folder to preserve connections and keys.
- `data/secret.key` is the master encryption key — do not share or delete it.
```

---

## Repository Configuration

### About Description (≤ 350 chars)
```
Lightweight portable SSH terminal for Windows. Multi-tab + split panes, SFTP browser, drag-and-drop upload, SSH key management, AES-256-GCM credential storage. Built with Go + Wails v2 + Vue 3 + xterm.js.
```

### Topics / Labels
```
ssh  sftp  terminal  wails  go  vue3  xterm  windows  desktop  portable
```

### Homepage URL
Leave blank until a dedicated site exists, or point to the GitHub repository itself.

---

## Version Roadmap

### v0.3.0 — Usability
- Replace `prompt()` / `confirm()` dialogs in SFTP panel with custom in-app modals
- SFTP recursive directory delete
- Configurable SSH connection timeout (currently hardcoded 15 s)
- Terminal search keyboard shortcut documentation in UI

### v0.4.0 — Connectivity
- ProxyJump / bastion host support
- Local port forwarding
- SSH agent forwarding

### v1.0.0 — Stability milestone
Prerequisites before calling this v1.0:
- [ ] At least one release cycle with no Critical/High bugs reported
- [ ] Unit tests for `cryptox`, `portable`, `config`, `keymgr`
- [ ] CI pipeline (GitHub Actions): `go vet` + `go build` on push
- [ ] macOS build exploration (Wails supports it; requires code-signing setup)
- [ ] CHANGELOG maintained through at least 2 release cycles
