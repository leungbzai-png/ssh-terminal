# Changelog

All notable changes to this project will be documented in this file.

## [0.4.0] - Unreleased

### Added
- **SSH KeepAlive**: periodic `keepalive@openssh.com` requests keep idle sessions and NAT mappings alive. New settings `keepAliveEnabled` (default on) and `keepAliveIntervalSec` (default 30 s, range 10–600 s) in the Settings dialog. The keepalive goroutine exits cleanly when the session closes and never blocks the stdout/stderr/wait goroutines.
- **Quick Connect**: connect to a host without saving it. Address / port / user / auth (password or external key file). The temporary password and passphrase live only in memory and are never written to `hosts.json`. An optional "记住此主机" checkbox saves the host through the existing AES-256-GCM encrypted path.
- **Import `~/.ssh/config`**: parse basic OpenSSH client config (`Host`, `HostName`, `User`, `Port`, `IdentityFile`) with a preview before import. Duplicate hosts (same address+port+user) are skipped, not overwritten. `~` is expanded; missing `IdentityFile` paths produce a warning without crashing. Complex directives (`Host *`, `Match`, `Include`, `ProxyJump`, forwards) are skipped with a warning. Imported `IdentityFile` keys are referenced by path only — no plaintext private key is ever copied into `data/`.
- New `internal/sshconfig` package with a pure, unit-tested OpenSSH config parser.

### Changed
- `sshsess.Manager.Open` now takes an additional `keepAliveSec int` parameter (0 disables keepalive). Internal API only.

## [0.3.0] - 2026-06-10

### Added
- Custom in-app SFTP dialogs: replaced browser `confirm()` and `prompt()` with `ConfirmDialog` and new `InputDialog` components — no more native OS dialogs
- SFTP recursive directory delete: directories now delete all contents recursively with a safety guard rejecting empty/root paths
- Configurable SSH connection timeout: new "连接超时" setting (default 15 s, range 5–120 s) in the Settings dialog; applies to all new connections including key deployment
- GitHub Actions CI: `go vet` + `go build` on Windows, frontend type-check + build on Ubuntu, runs on every push and pull request to `main`
- `InputDialog.vue` component: generic reusable text-input dialog used by SFTP mkdir and rename flows
- Per-tab SSH session close confirmation: clicking the tab X or "关闭标签" on an active/connecting session now shows a confirm dialog instead of closing immediately (shipped in v0.2.0 post-release patch, formally documented here)

### Fixed
- SFTP "删除" context menu item now correctly handles non-empty directories (previously failed with an SFTP error)
- SFTP mkdir and rename no longer block the UI thread via browser `prompt()`

## [0.2.0] - 2026

### Added
- Split-pane terminal layout (up to 4 panes)
- SFTP file browser with upload, download, rename, mkdir, delete
- Batch recursive directory upload with progress reporting
- Managed SSH keypair generation (Ed25519, RSA 2048/4096)
- Key deployment to remote hosts via `authorized_keys`
- Command broadcast bar (send to single tab or all tabs in pane)
- Per-host command history in command bar (up to 200 entries)
- Light / dark / system theme with CSS design tokens
- AES-256-GCM encrypted storage for passwords and passphrases
- First-connect host key fingerprint confirmation dialog
- Session reconnect overlay on disconnect

### Changed
- Portable data layout: all user data stored under `<exe-dir>/data/`

## [0.1.0] - 2025

### Added
- Initial SSH terminal with xterm.js (FitAddon, SearchAddon, WebLinksAddon)
- Host management with encrypted credential storage
- Known-hosts strict verification
- External SSH key file support
