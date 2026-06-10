# Changelog

All notable changes to this project will be documented in this file.

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
