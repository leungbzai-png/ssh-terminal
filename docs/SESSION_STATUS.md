# Session Status — SSH Terminal

**Last updated:** 2026-06-10  
**Updated by:** Claude Sonnet 4.6 (Phase 5–7 documentation pass)

---

## Current Version

| Field | Value |
|-------|-------|
| Version | **v0.2.0** |
| Git tag | `v0.2.0` |
| Branch | `main` |
| Latest commit | `43d06f6` — fix: restore dist gitkeep after build |

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
| v0.2.0 | 2026-06-10 | ✅ Published | `ssh-terminal-v0.2.0-windows-amd64.zip` (4.51 MB) |
| v0.1.0 | 2025 | Historical only (not on GitHub) | — |

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

---

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

### Immediate (v0.3.0)
1. Replace `prompt()`/`confirm()` in `SftpPanel.vue` with custom dialogs — most user-visible rough edge
2. SFTP recursive delete
3. Import hosts from `~/.ssh/config` — high value, commonly requested in similar tools
4. GitHub Actions CI — important for project health

### Medium Term (v0.4.0)
5. ProxyJump / bastion host support
6. Local port forwarding
7. Session keep-alive (ServerAliveInterval)

### Before v1.0.0
8. Unit tests for `cryptox`, `portable`, `config`, `keymgr`
9. CI pipeline stable
10. macOS build exploration

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
