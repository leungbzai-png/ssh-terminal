# Changelog

All notable changes to this project will be documented in this file.

## [1.2.1] - 2026-07-05

> **Release caveat (read this):** this UI-polish patch is **code-complete** and
> its **automated** gate passes (Go unit + build-tagged integration tests,
> frontend typecheck/build, Windows build), but the **manual Workspace Resize
> GUI QA** (`docs/WORKSPACE_RESIZE_QA.md`) was **NOT executed** before release —
> those cases remain NOT RUN. The resize **GUI** behavior (dragging, xterm
> reflow, persistence) should be treated as **release-caveated until
> user-tested**, mirroring v1.2.0/v1.1.0/v1.0.0. Those earlier GUI-QA items
> (VPS monitor, SFTP two-pane, auto-reconnect) also remain open.

### Changed — Resizable Workspace Splitters (UI polish)
- **Draggable splitters** between the **VPS monitor ↔ terminal** and the
  **terminal ↔ SFTP** panel, replacing the previous fixed/clamped column widths.
- **Monitor and SFTP panel widths are adjustable** by dragging; a splitter only
  appears for a panel that is open (a closed panel reclaims its space for the
  terminal).
- **Double-click a splitter resets** that panel to its default width.
- Widths clamp to safe min/max and **scale down on narrow windows** so the
  terminal keeps a usable minimum and the layout does not overflow.
- **Terminal reflows** after a resize through the terminal's existing
  `ResizeObserver` (no manual fit plumbing); drag is `requestAnimationFrame`-
  throttled.

### Persistence
- Width preferences persist **locally** via non-secret `localStorage` **integer
  pixel values only** (`ssh-terminal.monitorWidth`, `ssh-terminal.sftpWidth`).
  No local/remote paths, hostnames, IPs, usernames, credentials, monitor
  samples, or SFTP listings are stored. No `session.json` / `bookmarks.json` /
  secret-storage changes.

### Not changed
- No monitor metrics, no SFTP transfer logic, no SSH/security/storage model
  changes, no telemetry, and no release-artifact rules changed (Windows portable
  zip remains exe + README + LICENSE only).

## [1.2.0] - 2026-07-05

> **Release caveat (read this):** the VPS monitor is **code-complete** and its
> **automated** tests pass — the `internal/sysmon` parser/CPU-delta unit tests
> and a build-tagged `internal/sshsess` integration test that drives
> `Manager.Run` end to end (exec round-trip + `Run → sysmon.ParseAll` pipeline).
> However, the **manual VPS monitor GUI QA** (`docs/VPS_MONITOR_QA.md`) was
> **NOT executed** before this release — those cases remain NOT RUN. The live
> panel behavior (rendering, sparklines, interval switching, disconnect/
> unsupported states, timer cleanup) should be treated as **release-caveated
> until user-tested**, mirroring the posture v1.1.0/v1.0.0 shipped with. Open
> GUI-QA items from v1.1.0 (SFTP two-pane) and v1.0.0 (auto-reconnect) also
> remain unexecuted.

### Added — VPS Monitor Sidebar
- **Agentless, Linux-only VPS monitor** in a left-side, per-tab collapsible
  sidebar. While a tab is connected, it polls lightweight system metrics over
  the existing SSH connection and shows **CPU %**, **memory %**, **swap %** (when
  present), **disk usage for `/`**, **load average** (1/5/15m), and **uptime**,
  with real-time **sparklines** for CPU and memory.
- **One compact command per sample.** A single fixed command (`sysmon.Command`,
  no user/session interpolation) reads `/proc/stat`, `/proc/meminfo`,
  `/proc/loadavg`, `/proc/uptime`, `df -P /`, and `uname -s` in one round trip on
  a **separate SSH channel** (`sshsess.Manager.Run`), so monitoring never
  disturbs the interactive shell. CPU usage is a backend-computed delta between
  successive `/proc/stat` samples (`sysmon.Manager`, per session).
- **Per-tab controls.** Polling interval option (**2s / 5s / 10s**, default 5s)
  and an enable/disable toggle per tab (the panel doubles as the switch —
  monitoring runs only while the panel is shown).
- **Clear states.** Distinct UI for disconnected/no-session, unsupported host
  (non-Linux), loading, and error, separate from live metrics.
- New backend: `internal/sysmon` (parsers + CPU-delta manager),
  `sshsess.Manager.Run`, and the `MonitorSample` Wails bridge
  (`MonitorSnapshot`).

### Not in scope for v1.2.0
- No remote agent/daemon, no sudo, no remote install; no multi-server dashboard,
  historical database, Prometheus/Grafana, external monitoring API, or alerting;
  no process list/top clone, no per-interface network graph; no Windows/macOS/BSD
  **remote** host support; no metrics persistence.

### Security
- Monitoring is read-only and **agentless**: a fixed command with no
  session/user-derived interpolation (no command-injection surface), run only on
  a connected session. All samples (snapshots + sparkline history) live **in
  memory only and are never persisted** to disk — no `session.json` /
  `bookmarks.json` / secret-storage changes. Nothing is sent anywhere except the
  local UI.

## [1.1.0] - 2026-07-05

> **Release caveat (read this):** the SFTP two-pane feature is **code-complete**
> and its **automated** unit + build-tagged integration tests pass (including the
> `internal/localfs` browse API and the recursive `sftpx.DownloadPaths` /
> `Exists` backend). However, the **manual SFTP two-pane GUI QA**
> (`docs/SFTP_TWO_PANE_QA.md`) was **NOT executed** before this release — those
> cases remain NOT RUN. The GUI flows (pane rendering, drag-drop regression,
> overwrite dialogs, progress, and the `LocalParent` Wails multi-return) should
> be treated as **release-caveated until user-tested**. This mirrors the posture
> v1.0.0 shipped with for GUI auto-reconnect.

### Added — SFTP Two-Pane Foundation
- **Local/remote two-pane file browser.** The SFTP panel now shows a **local**
  pane beside the existing **remote** pane. The local pane browses the local
  filesystem (home, folders, drive roots on Windows) read-only, with
  up-navigation and refresh. Layout is responsive: side-by-side when wide,
  stacked when narrow.
- **Local → remote upload.** Select a local file or folder and upload it into
  the current remote directory (folders upload recursively via the existing
  batch upload).
- **Remote → local download (files and folders).** Select a remote file or
  folder and download it into the current local directory. Directory downloads
  are **recursive** (new `sftpx.DownloadPaths`).
- **Overwrite confirmation.** Before a two-pane transfer, the destination is
  checked (`SftpExists` / `LocalExists`); if the target already exists, a
  confirmation dialog offers overwrite or cancel. (Top-level name conflicts
  only; directory overwrite merges — deep per-file conflict resolution is out of
  scope.)
- New backend browse API (`internal/localfs`: List/Home/Roots/Parent/Exists) and
  its Wails bridge (`LocalList`/`LocalHome`/`LocalRoots`/`LocalParent`/
  `LocalExists`); `SftpDownloadPathsTracked` / `SftpExists` bridge methods.

### Preserved
- Existing remote actions are unchanged: remote upload, per-file download,
  mkdir, rename, delete-with-confirmation, remote-path bookmarks, and read-only
  text preview. The window **drag-drop upload** (`app:filedrop`) and its progress
  namespace are untouched.

### Not in scope for v1.1.0
- No transfer queue, no multi-threaded transfer, no resumable transfer, no
  background transfer manager, no multi-select, no advanced/persisted conflict
  resolution, and no file editor. These remain out of scope.

### Security
- The local pane is read-only browse + user-initiated transfers only. Local cwd,
  local listings, and selected local paths are held in memory and **never
  persisted** (no `session.json` / `bookmarks.json` / secret-storage changes).
  The safe host export rules and `data/secret.key` handling are unchanged.

## [1.0.0] - 2026-07-04

**Stable Release.** SSH Terminal 1.0.0 is the first stable release. It
consolidates the completed milestone work from the 0.x series and focuses on
stability, security, compatibility, and release readiness — **it does not add a
new feature scope**.

### Included milestone scope (unchanged behavior)
- **v0.4.0 Connection UX** — SSH KeepAlive, Quick Connect (secrets never
  persisted), import of `~/.ssh/config`.
- **v0.5.0 Host Management + Secure Storage** — host groups & search, safe host
  export/import (no secrets), encrypted private-key import (`.key.enc`).
- **v0.7.0 Terminal UX + SFTP UX** — terminal search/font controls, tab restore,
  shortcut help, SFTP transfer progress, remote bookmarks, read-only text
  preview.
- **v0.9.0 Advanced SSH + Hardening** — ProxyJump/bastion, local/remote/dynamic
  SOCKS5 forwarding, auto-reconnect, connection diagnostics, secret redaction,
  storage-compat hardening.

### Stabilization for 1.0.0
- **Version and documentation polish** — all version strings bumped to 1.0.0;
  README/CHANGELOG/roadmap/handoff updated.
- **Automated release gate** — `go test ./...`, `go vet ./...`, `go mod verify`,
  frontend build, and Windows build all pass.
- **Build-tagged backend-live Advanced SSH integration tests** (added after
  v0.9.0, `//go:build integration`) drive the real session manager against
  disposable in-process SSH servers on 127.0.0.1: ProxyJump/bastion,
  local/remote/dynamic-SOCKS forwarding, occupied-port handling, connection
  diagnostics, runtime cleanup, and the auto-reconnect backend close signal.
  Excluded from the normal test run; part of the v1.0.0 readiness gate.
- **Windows portable artifact verification** — the release zip contains only
  `ssh-terminal.exe`, `README.md`, and `LICENSE`.
- **Security / secret-storage regression checks** — no plaintext password,
  passphrase, or private key on disk; Quick Connect secrets not persisted; safe
  export excludes secrets; the release zip excludes `data/`, secrets, logs, and
  test artifacts.

### Security model (unchanged)
- Passwords and key passphrases are AES-256-GCM encrypted at rest with a
  locally-generated key; no plaintext secrets on disk; strict `known_hosts`
  verification; no telemetry, no auto-update, no network calls beyond
  user-initiated SSH/SFTP.

### Known limitation
- The **GUI auto-reconnect** cap/cancel/discriminator behavior (Vue-side) was
  **not separately human-tested** for this release. Its **backend close signal**
  is covered by the integration tests, and the frontend logic was reviewed with
  no new issue found, but full manual GUI QA of the reconnect UX remains
  outstanding. See `docs/QA_v0.8.0_v0.9.0.md` (section E).

### No product code changes
- 1.0.0 is documentation, versioning, and release packaging on top of the
  v0.9.0 code plus the post-v0.9.0 integration-test infrastructure. No feature
  or behavior of the shipped application changed.

## [0.9.0] - 2026-07-03

Part 3 — combined release of the planned **v0.8.0 Advanced SSH** and **v0.9.0
Hardening** scopes. Both were developed and QA'd together and ship in this
single pre-1.0 release. There is **no separate v0.8.0 tag or GitHub Release**;
the v0.8.0 Advanced SSH work is included here.

### Added — Advanced SSH (v0.8.0 scope, released as part of 0.9.0)
- **ProxyJump / bastion**: a saved host can connect through a single jump host, either by referencing another saved host (its encrypted credentials stay in secure storage and are used only at connect time) or by manual, **key-only** parameters. A manual bastion can never carry a password — password bastions must use the saved-host reference. The bastion connection is closed together with the session.
- **Local port forwarding**: 0..N per host (name, localHost, localPort, remoteHost, remotePort, enabled). Local bind defaults to 127.0.0.1; ports are validated 1–65535; duplicate local binds are rejected.
- **Remote port forwarding**: 0..N per host. Whether a non-loopback bind succeeds depends on the SSH server's `GatewayPorts` policy (surfaced in the UI); a server refusal is reported, not fatal.
- **Dynamic SOCKS5 forwarding**: 0..N per host — a local SOCKS5 proxy (e.g. 127.0.0.1:1080) tunnelled over SSH (no-auth + CONNECT; IPv4/IPv6/domain).
- **Auto reconnect**: per-host, capped (maxAttempts 0–10, delaySeconds 1–60). Triggers only on an unexpected drop after the session was established — never on a clean exit, a user close, or an authentication failure looping forever — and the user can cancel a pending reconnect.
- **Connection diagnostics**: connection failures are classified into readable categories (DNS / TCP refused-or-timeout / SSH handshake / auth / key-or-passphrase / proxy jump / port forward).

### Hardening (v0.9.0)
- **Storage compatibility**: v0.7.0 `hosts.json` (no `advanced` field) loads unchanged; missing/unknown fields never crash; a corrupt `hosts.json` reports an error instead of panicking and is never silently overwritten with an empty list.
- **Input validation**: Advanced SSH config is validated and defaulted in the backend (not only the UI) before it can persist or connect.
- **Runtime cleanup**: a per-session tunnel set owns every listener and in-flight forwarded connection; closing a tab / disconnecting / app exit releases all bound ports and closes the bastion connection.
- **Secret redaction**: a value-based redaction helper scrubs the actual host/bastion secrets from connection errors, and strips any embedded PEM private-key block from error/close-event payloads.

### Security
- No plaintext password, passphrase, or private key is stored. Quick Connect secrets are never persisted. The safe host export continues to exclude all secret material; the new Advanced SSH config it now carries is **non-secret** (a jump reference is only a host ID). ProxyJump and tunnel configuration store no secrets.

## [0.7.0] - 2026-07-03

Part 3 — combined release of the planned **v0.6.0 Terminal UX** and **v0.7.0 SFTP UX** scopes.
Both scopes were developed and QA'd together and ship in this single release. There is **no
separate v0.6.0 tag or GitHub Release**; the v0.6.0 Terminal UX work is included here.

### Added — SFTP UX
- **Upload/download progress**: the SFTP panel shows a footer progress bar for its own uploads and downloads (filename, direction, percentage). Failures surface an error and never crash the UI. Progress uses a dedicated `sftp:xfer:*` event namespace, kept separate from the window drag-upload flow.
- **Drag-upload polish**: the drop overlay now shows an accept state with the target remote directory, and a distinct reject state when there is no connected session (files only).
- **Remote path bookmarks**: per-host SFTP remote-path bookmarks (add current path, jump, delete), stored in `data/bookmarks.json`. Quick Connect tabs (no saved host) show a not-supported hint instead of failing.
- **Read-only text preview**: double-click a text file (or right-click → 预览) to preview it read-only. Files over 512 KiB report "too large" (download instead); non-UTF-8/binary files are refused. The remote file is never modified.

### Security — SFTP UX
- Bookmarks and restored-tab state contain only non-secret fields (name, path, host id/name). They are stored in their own files and are **not** part of the safe host export.
- Text preview is read-only and size-capped; it never writes to the remote file.

### Added — Terminal UX (v0.6.0 scope, released as part of 0.7.0)
- **Terminal search feedback**: the in-terminal search (Ctrl+F) now shows a live match count (index/total) and a 无匹配 indicator when nothing matches. Search remains per-tab and does not block input.
- **Font family presets**: the terminal font-family setting gains a datalist of common monospace fonts; a missing font falls back to the default without crashing.
- **Font size controls**: font size range widened to 8–32 (clamped). New shortcuts Ctrl+= / Ctrl+- adjust size and Ctrl+0 resets it; the setting persists.
- **Tab restore**: on launch the app restores the previous session's *saved-host* tabs as idle ("Ready to connect") without auto-connecting. Only a non-secret host reference (id + display name) is persisted, to `data/session.json`. Quick Connect tabs are never persisted; removed hosts are skipped on restore.
- **Keyboard shortcut help**: a new help dialog (sidebar button or F1) lists the real shortcuts and mouse actions.

### Security — Terminal UX
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
