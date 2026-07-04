# Session Status — SSH Terminal

**Last updated:** 2026-07-05  
**Updated by:** Claude Opus 4.8 (v1.1.0 SFTP two-pane — released with GUI-QA caveat)

---

## Current Version

| Field | Value |
|-------|-------|
| **Latest release** | **v1.1.0 (2026-07-05)** — SFTP Two-Pane Foundation (released with GUI-QA caveat) |
| Previous stable | **v1.0.0** (tag `v1.0.0`) — unchanged |
| Git tag | `v0.4.0`–`v1.0.0` unchanged; **`v1.1.0` created** at this final commit |
| Branch | `main` |
| Previous release commit | `83be738` (v1.0.0) |

---

## v1.1.0 — SFTP Two-Pane Foundation (RELEASED 2026-07-05, with GUI-QA caveat)

- **Scope (merged to `main` across commits 1–4):** local filesystem browse API
  (`internal/localfs`), recursive remote→local `sftpx.DownloadPaths`, the
  local/remote two-pane SFTP UI, and two-pane upload/download wiring with
  overwrite confirmation (`SftpExists`/`LocalExists`). Commit 5 bumped the
  version to 1.1.0; this final commit finalizes the release notes.
- **Automated release gate: green** — `go test ./...`, `go vet`, `go mod verify`,
  `go test -tags=integration ./...` (+ package-level), `npm run build`,
  `build-windows.bat`. localfs (List/Home/Roots/Parent/Exists) and the SFTP
  integration test (DownloadPaths + Exists) cover the backend paths.
- **⚠ Manual SFTP two-pane GUI QA: NOT executed** (`docs/SFTP_TWO_PANE_QA.md`,
  all cases still ☐/NOT RUN). The two-pane UI, drag-drop regression, overwrite
  dialogs, and the `LocalParent` Wails multi-return were **not** human-tested
  before release. **The user explicitly chose to release with this documented
  caveat** — the GUI flows should be treated as caveated until user-tested (same
  posture v1.0.0 shipped with for GUI auto-reconnect).
- **Released:** annotated tag `v1.1.0` + GitHub Release (Latest); artifact
  `ssh-terminal-v1.1.0-windows-portable.zip` (exe + README + LICENSE only). No
  QA case was marked PASS. v1.0.0 tag/release untouched.

---

## v1.0.0 — Stable Release (2026-07-04)

- **Scope:** stabilization only — version bump, docs/CHANGELOG/roadmap/handoff
  updates, release gate, Windows portable packaging. **No product code changed**
  (built on the v0.9.0 code plus the post-v0.9.0 build-tagged integration tests).
- **Release gate (all green):** `go test ./...`, `go vet ./...`, `go mod verify`,
  `go test -tags=integration ./...` (+ package-level, 3× stable), `npm run build`,
  `build-windows.bat`.
- **Artifact:** `ssh-terminal-v1.0.0-windows-portable.zip` (exe + README + LICENSE
  only), copied to `E:\Backup\Releases\`. Released as tag `v1.0.0` + GitHub
  Release (Latest). Older tags/releases unchanged; no v0.8.0 tag/release.
- **Feature freeze holds:** future 1.x is maintenance/bugfix only; v1.1.0 NOT
  started.
- **Honest QA note:** GUI auto-reconnect cap/cancel/discriminator (Vue-side) was
  **not separately human-tested**; its backend close signal is covered by the
  integration tests and the frontend logic was reviewed with no new issue found.
  Full manual GUI QA of the reconnect UX remains the one outstanding item
  (`docs/QA_v0.8.0_v0.9.0.md` section E, all items still ☐).

## Pre-v1.1.0 readiness pass (2026-07-04)

- Re-ran the full gate against the v1.0.0 baseline (`83be738`): `go test ./...`,
  `go vet`, `go mod verify`, `go test -tags=integration ./...` (+ package-level
  verbose), `npm run build`, `build-windows.bat` — **all green**. Security scan
  clean; working tree clean; no forbidden files tracked; `build-windows.bat`
  builds only the exe (no release zip created, nothing uploaded).
- **No product code bug found.** Auto-reconnect frontend logic reviewed again and
  confirmed correct (unexpected-drop starts a capped burst; user-close/clean-exit
  do not; cap/cancel/manual-supersede/success-reset all correct). **No v1.0.1
  needed.**
- Added a dedicated **manual QA checklist**: `docs/GUI_AUTO_RECONNECT_QA.md`
  (authored, **not yet human-executed**; no case marked PASS). This is the one
  remaining pre-expansion gap. Automated GUI coverage was intentionally not added
  (no frontend test runner; would require a component refactor out of scope here).
- **Ready to begin v1.1.0 feature iteration** once the user approves. This pass
  changed docs only (checklist + pointers); the commit is local and **not pushed
  / not tagged / not released** — awaiting explicit instruction.

---

## GitHub Repository

| Field | Value |
|-------|-------|
| URL | https://github.com/leungbzai-png/ssh-terminal |
| Visibility | Public |
| License | MIT |
| Topics | ssh, sftp, terminal, wails, go, vue3, xterm, windows, desktop, portable |

---

## Release Status

| Release | Date | Status | Artifacts |
|---------|------|--------|-----------|
| v0.9.0 | 2026-07-03 | ✅ Published (latest) — combined v0.8.0 Advanced SSH + v0.9.0 Hardening | `ssh-terminal-v0.9.0-windows-portable.zip` |
| v0.7.0 | 2026-07-03 | ✅ Published — combined v0.6.0 Terminal UX + v0.7.0 SFTP UX | `ssh-terminal-v0.7.0-windows-portable.zip` |
| v0.5.0 | 2026-07-02 | ✅ Published | `ssh-terminal-v0.5.0-windows-amd64.zip` (4,754,440 bytes ≈ 4.53 MB) |
| v0.4.0 | 2026-07-02 | ✅ Published | `ssh-terminal-v0.4.0-windows-amd64.zip` (4,745,983 bytes ≈ 4.53 MB) |
| v0.3.0 | 2026-06-10 | ✅ Published | `ssh-terminal-v0.3.0-windows-amd64.zip` |
| v0.2.0 | 2026-06-10 | ✅ Published | `ssh-terminal-v0.2.0-windows-amd64.zip` (4.51 MB) |
| v0.1.0 | 2025 | Historical only (not on GitHub) | — |

**v0.5.0 release:** annotated tag `v0.5.0` on `main`. GitHub Release published and marked latest:
https://github.com/leungbzai-png/ssh-terminal/releases/tag/v0.5.0
Uploaded artifact: `ssh-terminal-v0.5.0-windows-amd64.zip` (4,754,440 bytes). Manual QA checklist A–I passed; git status clean after release. `v0.4.0` tag unchanged.

**v0.4.0 release:** tag `v0.4.0` points to commit `3b09cfc8ebb35c58761da56b1a1111defdfb3c22`.
GitHub Release published: https://github.com/leungbzai-png/ssh-terminal/releases/tag/v0.4.0
Uploaded artifact: `ssh-terminal-v0.4.0-windows-amd64.zip` (4,745,983 bytes). Manual QA checklist A–G passed; git status clean after release.

Release zip location (local backup): `E:\Backup\Releases\ssh-terminal-v0.2.0-windows-amd64.zip`

---

## Completed Work

### Phase 1 — Audit (completed 2026-06-10)
- Full read-only code audit of all Go, Vue, TypeScript, config files
- Identified 4 Critical, 5 High, 8 Medium, 7 Low issues
- No code changes

### Phase 2 — Engineering (completed 2026-06-10)
- **Batch A**: `.gitignore` hardened, zip artifact deleted, `.gitkeep` added, `LICENSE` + `CHANGELOG.md` created
- **Batch B**: Build scripts de-localized (removed `GOROOT=D:\go`), Wails CLI version unified to `v2.12.0`, `README.md` fully rewritten
- **Batch C**: Module renamed to `github.com/leungbzai-png/ssh-terminal`, version unified to `0.2.0`, `wails.json` author/copyright updated, 15 import paths updated across 6 Go files, `go build ./...` verified
- **Batch D**: Fixed drag-drop upload (`onFileDrop` now registered via `runtime.OnFileDrop` + `DragAndDrop` option), fixed `TabBar.vue` event listener leak
- **Batch E**: Deleted `dirty` dead field from `hosts.go`, deleted `stdout`/`stderr` dead fields from `Session` struct, fixed `useTheme.ts` module-level singleton, added `buildAuthForDeploy` TODO comment
- **Batch F**: `docs/architecture.md` generated from source

### Phase 3 — Git Preparation (completed 2026-06-10)
- `.gitignore` fixed (`frontend/dist/*` vs `frontend/dist/`, added `.claude/`)
- `docs/GITHUB_RELEASE.md` created
- `go build ./...`, `go vet ./...`, `go mod verify` all pass
- Git initialized, initial commit made, pushed to GitHub

### Phase 4 — Windows Build & Release (completed 2026-06-10)
- `build-windows.bat` executed successfully (Wails v2.12.0 auto-installed)
- `build\bin\ssh-terminal.exe` generated: 11.37 MB
- Release zip created: `ssh-terminal-v0.2.0-windows-amd64.zip` (4.51 MB)
- `build-windows.bat` patched to restore `frontend/dist/.gitkeep` after Vite clean build
- GitHub Release v0.2.0 published

### Phase 5–7 — QA, Roadmap, Maintenance Docs (completed 2026-06-10)
- `docs/QA_CHECKLIST.md` — functional test cases for all features
- `docs/ROADMAP.md` — competitive analysis + top 20 features + v0.3.0–v1.0.0 plan
- `docs/AI_HANDOFF.md` — architecture guide for AI/new maintainers
- `docs/PROJECT_CONTEXT.md` — design decision rationale
- `docs/SESSION_STATUS.md` — this file
- `docs/RELEASE_PROCESS.md` — step-by-step release guide

### Post-v0.2.0 Patch — Tab Close Confirmation (completed 2026-06-10)
- Created `frontend/src/components/ConfirmDialog.vue` — generic reusable confirm dialog
- Modified `frontend/src/components/TabBar.vue` — per-tab close confirmation for active sessions
- `go build ./...` and frontend build both verified passing
- Windows zip rebuilt: `ssh-terminal-v0.2.0-windows-amd64.zip` (4.51 MB, includes feature)

### v0.3.0 — Usability & Polish (completed 2026-06-10)
- Created `frontend/src/components/InputDialog.vue` — generic reusable text-input dialog
- Modified `frontend/src/components/SftpPanel.vue` — all `confirm()`/`prompt()` replaced with `ConfirmDialog`/`InputDialog`; recursive delete wired for dirs
- Modified `internal/sftpx/sftpx.go` — added `DeleteRecursive` using `sftp.Client.RemoveAll` with empty/root safety guard
- Modified `app.go` — added `SftpDeleteRecursive` API, reads `ConnectTimeoutSec` from settings for both `OpenSession` and `DeployPublicKeyToHost`
- Modified `internal/sshsess/manager.go` — `Open()` accepts `timeoutSec int` parameter
- Modified `internal/config/config.go` — added `ConnectTimeoutSec int` (default 15)
- Modified `frontend/src/wails.d.ts` — added `SftpDeleteRecursive`, `connectTimeoutSec`
- Modified `frontend/src/components/SettingsDialog.vue` — added "连接超时（秒）" field
- Created `.github/workflows/ci.yml` — Go build+vet (Windows) + frontend build (Ubuntu)

---

### v0.4.0 — Part 1: Connection UX (in development 2026-07-02)
- Restructured `docs/ROADMAP.md` into 3 parts (v0.4.0 / v0.5.0 / v0.6.0–v1.0.0)
- **SSH KeepAlive**: `config.Settings` gained `KeepAliveEnabled` (default true) + `KeepAliveIntervalSec` (default 30); `sshsess.Manager.Open` gained a `keepAliveSec` param and a `done`-channel-driven keepalive goroutine sending `keepalive@openssh.com`; `SettingsDialog.vue` + settings store + `wails.d.ts` updated
- **Quick Connect**: new `SshOpenQuick` API + `QuickConnectParams` (in-memory only, never persisted); `QuickConnectDialog.vue`; `sessions` store `quickParams` map + `openQuickInActivePane` (cleared in `closeTab`); `Terminal.vue` branches quick vs saved; "记住此主机" reuses `UpsertHost`
- **Import `~/.ssh/config`**: new `internal/sshconfig` package (pure parser + unit tests); `DefaultSshConfigPath`/`PickSshConfig`/`PreviewSshConfig`/`ImportSshConfig` APIs; `ImportConfigDialog.vue` with preview, duplicate/missing-key badges; external IdentityFile referenced by path only
- `Sidebar.vue` footer: 快速连接 / 新增主机 / 导入配置; `App.vue` wiring; version → 0.4.0 in `app.go`, `wails.json`, `frontend/package.json`
- **Final verification passed**: `go build ./...`, `go vet ./...`, `go test ./...`, `frontend npm run build`, `build-windows.bat` all pass
- **Manual QA checklist A→G passed** (see `qa-local/MANUAL_QA_v0.4.0.md`, not tracked): fresh startup, KeepAlive default/range/idle-survival, Quick Connect no-save + remember (encrypted), config import (skip `Host *`, ProxyJump/missing-key warnings, duplicate-safe), security scans clean
- **Release artifact prepared**: `ssh-terminal-v0.4.0-windows-amd64.zip` (exe + README + LICENSE only)

### v0.5.0 — Part 2: Host Management + Secure Storage (released 2026-07-02)
- **Host groups**: `Host.Group` (already in schema) formalized in UI; `Sidebar.vue` groups by `group || "Ungrouped"` (Ungrouped sorts last); `HostDialog.vue` group field gains a `<datalist>` of existing groups. No backend/schema change.
- **Host search**: existing `Sidebar.vue` search verified against acceptance (alias/address/user/group, case-insensitive, hides empty groups, clears to full).
- **Safe host export/import**: new `internal/hosts/export.go` (`SafeHost` whitelist struct, `BuildExport`/`MarshalExport`/`ParseExport`, format `ssh-terminal.hosts.safe-export` v1); `app.go` APIs `ExportHosts` / `PreviewHostsImport` / `ImportHosts(entries, overwrite)`; new `ImportHostsDialog.vue`; Sidebar "导出主机"/"导入主机" buttons; App.vue wiring. Duplicates (address+port+user) skip by default, overwrite behind explicit checkbox; new hosts get fresh IDs.
- **Encrypted private-key import**: new `internal/keymgr/import.go` (`ImportFromFile` — reads key on Go side, validates, encrypts original bytes to `.key.enc`, passphrase transient/never persisted); `app.go` `ImportPrivateKey`; Keys dialog "导入已有私钥" section.
- **Security enforcement**: `internal/hosts/export_test.go` + `internal/keymgr/import_test.go` + `app_test.go` — sentinel + PEM-marker scans on generated artifacts only, encrypted-field assertions, no-plaintext-key-file assertions, passphrase-not-persisted, and host-import dedup/fresh-ID/overwrite-preserves-secret.
- **Version bumped to 0.5.0**: `app.go` AppInfo, `wails.json` (+author/copyright → noobra2), `frontend/package.json`, `frontend/package-lock.json`.
- **Automated verification (all pass)**: `go build ./...`, `go vet ./...`, `go test ./...`, `go mod verify`, `frontend npm run build` (vue-tsc + vite), `build-windows.bat` (exe built ~11.4 MB).
- **QA build**: `qa-local/ssh-terminal-v0.5.0-qa/ssh-terminal.exe` + empty `data/`. Checklist: `qa-local/MANUAL_QA_v0.5.0.md` (A–I) — **passed**. qa-local/ is git-ignored.
- **Released**: annotated tag `v0.5.0` pushed; GitHub Release published. `v0.4.0` tag unchanged.

### v0.6.0 + v0.7.0 — Terminal & SFTP UX (released 2026-07-03 as v0.7.0, bundled)
- **Terminal search**: `Terminal.vue` search gains live match count + 无匹配 (SearchAddon.onDidChangeResults; live incremental).
- **Font controls**: `settings` store `bumpFontSize`/`resetFontSize` (clamp 8–32); Ctrl +/-/0 in `Terminal.vue`; `SettingsDialog` font datalist + range 8–32.
- **Tab restore**: new `internal/session` (`data/session.json`, non-secret hostId+hostName); `GetOpenTabs`/`SaveOpenTabs`; `sessions` store `idle` status + `openSavedTabIdle` + debounced persist; `Terminal.vue` idle overlay; `App.vue restoreTabs` skips missing hosts; Quick tabs never persisted.
- **Shortcut help**: new `ShortcutHelpDialog.vue`; F1 + sidebar button.
- **Transfer progress**: `sftpx.DownloadWithProgress`; `app.go` `SftpDownloadTracked`/`SftpUploadTracked` on dedicated `sftp:xfer:*` events; `SftpPanel` footer progress bar.
- **Drag polish**: `App.vue` overlay accept/reject + target dir.
- **Remote bookmarks**: new `internal/bookmarks` (`data/bookmarks.json`, non-secret); `ListBookmarks`/`AddBookmark`/`DeleteBookmark`; `SftpPanel` bookmark menu.
- **Text preview**: `sftpx.IsProbablyText` + `ReadFilePreview`; `app.go SftpPreviewText` (512 KiB cap, read-only); `TextPreviewDialog.vue`.
- **Tests**: `internal/bookmarks/bookmarks_test.go`, `internal/session/session_test.go`, `internal/sftpx/sftpx_test.go` (+ existing v0.5.0 tests). All `go test ./...` pass.
- **Version bumped to 0.7.0**: `app.go`, `wails.json`, `frontend/package.json` + lock.
- **Automated verification**: `go build/vet/test`, `go mod verify`, `npm run build`, `build-windows.bat` all pass.
- **QA build**: `qa-local/ssh-terminal-v0.6.0-v0.7.0-qa/`; checklist `docs/QA_v0.6.0_v0.7.0.md`.
- **Released** as combined **v0.7.0** on 2026-07-03: annotated tag `v0.7.0` on `main`, GitHub Release published (latest), artifact `ssh-terminal-v0.7.0-windows-portable.zip` (exe + README + LICENSE only). **No separate v0.6.0 tag/Release.** `v0.4.0` / `v0.5.0` tags unchanged.

## Known Issues (Open)

| ID | Severity | Description | File | Planned Fix |
|----|----------|-------------|------|-------------|
| KI-01 | Medium | SFTP dialogs use browser `prompt()`/`confirm()` | `SftpPanel.vue` | v0.3.0 |
| KI-02 | Medium | SFTP cannot delete non-empty directories | `internal/sftpx/sftpx.go` | v0.3.0 |
| KI-03 | Low | `known_hosts` may have duplicate entries per host | `sshsess/manager.go:appendKnownHost` | v0.4.0 |
| KI-04 | Low | TTY baud rate hardcoded at 14400 | `sshsess/manager.go` | v0.4.0 |
| KI-05 | Low | UI language mixing (Chinese + some English) | Various Vue components | v0.3.0 |
| KI-06 | Low | `buildAuthForDeploy` duplicates `buildAuth` | `app.go` + `sshsess/manager.go` | v0.4.0 |
| KI-07 | Low | 80ms `time.Sleep` in `ConfirmQuit()` | `app.go` | v0.4.0 |
| KI-08 | Low | `Manager.Open` reconnect doesn't close prior live client | `sshsess/manager.go` | v0.4.0 |

---

## Next Development Direction

### v0.4.0 — Part 1: Connection UX — ✅ released
- KeepAlive, Quick Connect, Import `~/.ssh/config` — implemented, QA passed, tagged `v0.4.0`

### v0.5.0 — Part 2: Host Management + Secure Storage — ✅ released 2026-07-02
1. Host groups / folders — done
2. Host search (formalized) — done
3. Safe host export/import (no secrets by default) — done
4. Encrypted private-key import (`.key.enc`) — done
5. Security-policy enforcement / no plaintext secrets on disk — done (tests + docs)

Manual QA A–I passed; tag `v0.5.0` + GitHub Release published.

### v0.6.0 + v0.7.0 — Part 3 (Terminal UX + SFTP UX) — ✅ released 2026-07-03 as v0.7.0
- Combined release under a single `v0.7.0` tag; no separate v0.6.0 tag/Release.

### v0.8.0 + v0.9.0 — Advanced SSH + Hardening — ✅ released 2026-07-03 as v0.9.0
- New Go packages: `internal/redact` (value-based secret scrubbing). New files in `internal/sshsess`: `tunnel.go` (local/remote/dynamic SOCKS forwards with a per-session tunnel set), `socks.go` (SOCKS5 CONNECT parsing), `diag.go` (error classification). `manager.go` refactored to `Open(OpenOptions)` with single-level ProxyJump (bastion client tracked + closed with the session).
- `internal/hosts/advanced.go`: non-secret `AdvancedSSH` (ProxyJump / forwards / auto-reconnect) with `Normalize()` validation; `Host.Advanced` pointer (omitempty → v0.7.0 data loads unchanged); included in `SafeHost` export. Corrupt-`hosts.json` hardening (error, no panic, no silent overwrite).
- `app.go`: jump-host resolution (saved-host secrets or key-only manual), tunnel-status events (`ssh:tunnel:<id>`), redacted + diagnosed connect errors. Version → 0.9.0.
- Frontend: HostDialog collapsed "高级 SSH" panel; Terminal auto-reconnect burst (capped, unexpected-drop only, cancellable) + tunnel status; `wails.d.ts` Advanced SSH / TunnelStatus types.
- Tests: `internal/hosts/advanced_test.go` + `compat_test.go`, `internal/redact/redact_test.go`, `internal/sshsess/diag_test.go` + `socks_test.go`, `advanced_app_test.go` (connect-error redaction, jump resolution, advanced export safety). All `go test ./...` + `go vet` pass; `npm run build` + `build-windows.bat` pass. `go test -race` not run (no gcc/CGO here).
- **Released** as combined **v0.9.0**: annotated tag `v0.9.0`, GitHub Release (latest), artifact `ssh-terminal-v0.9.0-windows-portable.zip`. **No separate v0.8.0 tag/Release.** v0.4.0 / v0.5.0 / v0.7.0 tags unchanged. Manual QA checklist: `docs/QA_v0.8.0_v0.9.0.md`.

### Post-v0.9.0 — Advanced SSH integration tests (2026-07-04, no release)
- The temporary `qa-local/sshqa/` backend-live QA harness was converted into a **committed, build-tagged Go integration suite** under `internal/sshsess/` (`integration_test.go` + `integration_server_test.go`, both `//go:build integration`).
- Covers ProxyJump/bastion, local/remote/dynamic-SOCKS forwarding, occupied-port resilience, connection diagnostics (TCP/auth/DNS/key/proxyjump), runtime cleanup (`CloseAll`→0, listener release), and the auto-reconnect close signal (unexpected-drop vs user-close). Disposable in-process SSH servers on 127.0.0.1; no real VPS, no committed secrets, all credentials runtime-generated, errors redacted.
- Excluded from `go test ./...`; run with `go test -tags=integration ./...` (or `go test -tags=integration ./internal/sshsess -run Integration -v`). Docs: `docs/INTEGRATION_TESTS.md`.
- **No release/tag/push/version bump.** v0.9.0 remains latest; **v1.0.0 not started.** Use this suite as a v1.0.0 readiness gate.
- **Project is in feature freeze** until v1.0.0: only stabilization, tests, and docs — no new product features. **v1.0.0 readiness gate** = `go test ./...` + `go test -tags=integration ./...` + `go vet ./...` + `go mod verify` + `npm run build` + `build-windows.bat` all green. Only remaining pre-v1.0.0 manual item: **GUI auto-reconnect cap/cancel/discriminator** (Vue-side; not covered by the backend-live integration tests).

### Next: v1.0.0 — Stable Release — do not start unprompted
- Stabilization + release cycle only; no new major feature. **Not started.**

---

## Tech Stack Summary

| Layer | Technology | Version |
|-------|-----------|---------|
| Desktop framework | Wails v2 | v2.12.0 |
| Backend language | Go | 1.23.4 (dev), requires ≥ 1.22 |
| SSH library | golang.org/x/crypto/ssh | v0.33.0 |
| SFTP library | github.com/pkg/sftp | v1.13.6 |
| Frontend framework | Vue 3 + Pinia | 3.5.12 / 2.2.4 |
| Terminal emulator | @xterm/xterm | 5.5.0 |
| Build tool | Vite | 5.4.10 |
| OS support | Windows 10/11 (WebView2) | — |

---

## File Structure Quick Reference

```
ssh-terminal/
├── main.go              Wails entry, DragAndDrop options
├── app.go               Go↔JS API bridge (404 lines)
├── internal/
│   ├── portable/        Path resolution (exe-relative)
│   ├── cryptox/         AES-256-GCM encryption
│   ├── config/          settings.json
│   ├── hosts/           hosts.json (passwords encrypted)
│   ├── keymgr/          SSH keypair management
│   ├── sshsess/         SSH sessions + PTY
│   └── sftpx/           SFTP operations
├── frontend/src/
│   ├── App.vue          Root, file drop handler
│   ├── stores/          Pinia: settings, hosts, sessions
│   ├── components/      UI components (12 files)
│   ├── composables/     useTheme (module singleton)
│   └── wails.d.ts       TypeScript types for Go API
└── docs/
    ├── architecture.md  Data flows + module contracts
    ├── AI_HANDOFF.md    This guide's companion
    ├── PROJECT_CONTEXT.md Design decisions
    ├── QA_CHECKLIST.md  Test cases
    ├── ROADMAP.md       Feature planning
    ├── RELEASE_PROCESS.md How to release
    └── SESSION_STATUS.md This file
```
