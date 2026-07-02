# Session Status — SSH Terminal

**Last updated:** 2026-07-02  
**Updated by:** Claude Opus 4.8 (v0.4.0 Part 1 — Connection UX)

---

## Current Version

| Field | Value |
|-------|-------|
| Version | **v0.4.0** |
| Git tag | `v0.4.0` → points to `3b09cfc` |
| Branch | `main` |
| Final release commit | `3b09cfc8ebb35c58761da56b1a1111defdfb3c22` (`3b09cfc`) |

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
| v0.4.0 | 2026-07-02 | ✅ Published (latest) | `ssh-terminal-v0.4.0-windows-amd64.zip` (4,745,983 bytes ≈ 4.53 MB) |
| v0.3.0 | 2026-06-10 | ✅ Published | `ssh-terminal-v0.3.0-windows-amd64.zip` |
| v0.2.0 | 2026-06-10 | ✅ Published | `ssh-terminal-v0.2.0-windows-amd64.zip` (4.51 MB) |
| v0.1.0 | 2025 | Historical only (not on GitHub) | — |

**v0.4.0 release:** tag `v0.4.0` points to commit `3b09cfc8ebb35c58761da56b1a1111defdfb3c22`.
GitHub Release published and marked latest: https://github.com/leungbzai-png/ssh-terminal/releases/tag/v0.4.0
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

### Next: v0.5.0 — Part 2: Host Management + Secure Storage
1. Host groups / folders
2. Host search (formalize)
3. Safe host export/import (no secrets by default)
4. Encrypted private-key import (`.key.enc`)
5. Security-policy enforcement / no plaintext secrets on disk

### Later (Part 3: v0.6.0 → v1.0.0)
- Terminal UX (v0.6.0), SFTP UX (v0.7.0), Advanced SSH incl. ProxyJump/forwarding/SOCKS (v0.8.0), Hardening + tests (v0.9.0), stable tag (v1.0.0)

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
