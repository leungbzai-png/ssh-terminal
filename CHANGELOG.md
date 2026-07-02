# Changelog

All notable changes to this project will be documented in this file.

## [0.7.0] - Unreleased (pending QA)

Part 3 — SFTP UX enhancements. Bundled with v0.6.0 in one QA build. **Not tagged or released.**

### Added
- **Upload/download progress**: the SFTP panel shows a footer progress bar for its own uploads and downloads (filename, direction, percentage). Failures surface an error and never crash the UI. Progress uses a dedicated `sftp:xfer:*` event namespace, kept separate from the window drag-upload flow.
- **Drag-upload polish**: the drop overlay now shows an accept state with the target remote directory, and a distinct reject state when there is no connected session (files only).
- **Remote path bookmarks**: per-host SFTP remote-path bookmarks (add current path, jump, delete), stored in `data/bookmarks.json`. Quick Connect tabs (no saved host) show a not-supported hint instead of failing.
- **Read-only text preview**: double-click a text file (or right-click → 预览) to preview it read-only. Files over 512 KiB report "too large" (download instead); non-UTF-8/binary files are refused. The remote file is never modified.

### Security
- Bookmarks and restored-tab state contain only non-secret fields (name, path, host id/name). They are stored in their own files and are **not** part of the safe host export.
- Text preview is read-only and size-capped; it never writes to the remote file.

## [0.6.0] - Unreleased (pending QA)

Part 3 — Terminal UX. Bundled with v0.7.0 in one QA build. **Not tagged or released.**

### Added
- **Terminal search feedback**: the in-terminal search (Ctrl+F) now shows a live match count (index/total) and a 无匹配 indicator when nothing matches. Search remains per-tab and does not block input.
- **Font family presets**: the terminal font-family setting gains a datalist of common monospace fonts; a missing font falls back to the default without crashing.
- **Font size controls**: font size range widened to 8–32 (clamped). New shortcuts Ctrl+= / Ctrl+- adjust size and Ctrl+0 resets it; the setting persists.
- **Tab restore**: on launch the app restores the previous session's *saved-host* tabs as idle ("Ready to connect") without auto-connecting. Only a non-secret host reference (id + display name) is persisted, to `data/session.json`. Quick Connect tabs are never persisted; removed hosts are skipped on restore.
- **Keyboard shortcut help**: a new help dialog (sidebar button or F1) lists the real shortcuts and mouse actions.

### Security
- Tab restore stores only a host reference — never a password, passphrase, private key, Quick Connect secret, terminal buffer, or SFTP state.

## [0.5.0] - 2026-07-02

Part 2 — Host Management + Secure Storage.

### Added
- **Host groups**: hosts can be organized into named groups. The sidebar renders one section per group; hosts with no group appear under a virtual **Ungrouped** section (always shown last). The host create/edit dialog has a group field with autocomplete of existing groups. Empty group is treated as Ungrouped. (`Host.Group` already existed in the schema; this formalizes the UX and default naming.)
- **Host search**: the sidebar search box filters hosts by alias, hostname, username, and group (case-insensitive). Groups with no matches are hidden; clearing the box restores the full grouped list. Search is frontend-only and never mutates host data.
- **Safe host export**: "导出主机" writes a JSON backup containing only non-secret host metadata (name, address, port, user, auth type, group, note, and key *references*). The file is explicitly labelled "安全，不含密码或私钥". It never contains a password, passphrase, encrypted secret, or private-key material. Format: `{"format":"ssh-terminal.hosts.safe-export","version":1,...}`.
- **Safe host import**: "导入主机" reads a safe-export file, shows a preview with duplicate and missing-key-path badges, and imports selected hosts. Duplicates (address+port+user) default to **skip**; overwriting requires an explicit checkbox. Imported hosts carry no passwords — the user adds credentials afterward. New hosts receive freshly minted IDs. Invalid/unknown files produce a friendly error and never corrupt the existing host list.
- **Encrypted private-key import**: the Keys dialog can import an existing private key file into the managed key store. The key is read on the Go side only, validated, and immediately encrypted to `data/keys/<id>.key.enc`. No plaintext private key is ever written under `data/`. A passphrase, if supplied, is used only to validate a protected key and is never persisted; the original protection is preserved at rest.

### Security
- New whitelist-based export struct (`hosts.SafeHost`) guarantees no secret field can leak into an export — it is built independently of the internal `Host`/`storedHost` types.
- Imported private keys are encrypted immediately; the passphrase is transient and never stored. `HasPassword` metadata reflects the key's real protection.
- Added automated tests asserting: exports contain no plaintext secrets (unique-sentinel scan + PEM-marker scan), `hosts.json` stores only encrypted secret fields, imported keys produce only `.key.enc` (no plaintext key file / no PEM markers under `data/keys`), passphrases are never persisted, and host import deduplicates (skip-by-default), assigns fresh IDs, and preserves the existing encrypted password on overwrite.
- `data/secret.key` format/location, and the `encPassword`/`encPassphrase` field contract, are unchanged.

## [0.4.0] - 2026-07-02

### Added
- **SSH KeepAlive**: periodic `keepalive@openssh.com` requests keep idle sessions and NAT mappings alive. New settings `keepAliveEnabled` (default on) and `keepAliveIntervalSec` (default 30 s, range 10–600 s) in the Settings dialog. The keepalive goroutine exits cleanly when the session closes and never blocks the stdout/stderr/wait goroutines.
- **Quick Connect**: connect to a host without saving it. Address / port / user / auth (password or external key file). The temporary password and passphrase live only in memory and are never written to `hosts.json`. An optional "记住此主机" checkbox saves the host through the existing AES-256-GCM encrypted path.
- **Import `~/.ssh/config`**: parse basic OpenSSH client config (`Host`, `HostName`, `User`, `Port`, `IdentityFile`) with a preview before import. Duplicate hosts (same address+port+user) are skipped, not overwritten. `~` is expanded; missing `IdentityFile` paths produce a warning without crashing. Complex directives (`Host *`, `Match`, `Include`, `ProxyJump`, forwards) are skipped with a warning. Imported `IdentityFile` keys are referenced by path only — no plaintext private key is ever copied into `data/`.
- New `internal/sshconfig` package with a pure, unit-tested OpenSSH config parser.

### Changed
- `sshsess.Manager.Open` now takes an additional `keepAliveSec int` parameter (0 disables keepalive). Internal API only.

### Security
- Quick Connect temporary passwords and passphrases are never persisted — they exist only in memory for the session and are dropped when the tab closes.
- Saved host passwords/passphrases continue to use AES-256-GCM encrypted storage (`encPassword`/`encPassphrase`); no plaintext secret is ever written to `hosts.json`.
- Imported `IdentityFile` entries are referenced by path only; no plaintext private key is copied into `data/`.
- `data/secret.key` format and location are unchanged.

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
