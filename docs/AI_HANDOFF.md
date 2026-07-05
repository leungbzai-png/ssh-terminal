# AI Handoff Guide — SSH Terminal

This document is for AI assistants (or new human maintainers) picking up work on this project.
Read this before touching any code. It was written after a complete audit + engineering pass.

---

## Project in One Paragraph

SSH Terminal is a portable Windows desktop SSH client built with Go + Wails v2 (WebView2 backend)
and a Vue 3 + TypeScript frontend. All user data lives next to the exe in `data/`. Credentials are
encrypted with AES-256-GCM. The app has zero external network calls beyond user-initiated SSH/SFTP.
It is personal-developer-scale: no database, no server, no tests (yet). CI via GitHub Actions added in v0.3.0.

**In progress: v1.2.1 — Resizable Workspace Splitters** (UI-polish patch, release prep on `main`). Draggable splitters between the VPS monitor ↔ terminal and terminal ↔ SFTP panel replace the old fixed/clamped grid columns. Lives in `frontend/src/components/PaneView.vue` (inline `gridTemplateColumns` computed from user-set widths; a splitter renders only for an open panel; rAF-throttled pointer drag; double-click resets a panel; widths clamp + scale down on narrow windows to protect the terminal minimum and avoid overflow; terminal reflow via the terminal's existing `ResizeObserver` — no manual fit). Widths persist through `frontend/src/composables/useWorkspaceLayout.ts`, a module-singleton storing only two non-secret integer px values in `localStorage` (`ssh-terminal.monitorWidth` / `ssh-terminal.sftpWidth`). No monitor/SFTP/SSH/storage change. Automated gate green; **manual `docs/WORKSPACE_RESIZE_QA.md` NOT executed** — caveated until human-tested. **Do not mark any WORKSPACE_RESIZE_QA case PASS without a real human run.**

**Released: v1.2.0 — VPS Monitor Sidebar** (2026-07-05 — tag `v1.2.0`, GitHub Latest; released **with a documented GUI-QA caveat**). An agentless, **Linux-only** left-side per-tab monitor that polls CPU / memory / swap / disk `/` / load / uptime over the existing SSH connection and draws CPU & memory sparklines. Backend: `internal/sysmon` (pure parsers for `/proc/stat`, `/proc/meminfo`, `df -P /`, `/proc/loadavg`, `/proc/uptime`, `uname -s` + a per-session CPU-delta `Manager`), `sshsess.Manager.Run` (one-off exec on a **separate** SSH channel — never the shell PTY), and the `App.MonitorSample` bridge (`MonitorSnapshot`). Frontend: `MonitorSidebar.vue` + `Sparkline.vue`, per-tab `showMonitor` toggle, and store-backed per-tab monitor state (interval/snapshot/error/trend, in-memory only). The automated gate (incl. a build-tagged `Manager.Run` integration test) is green, but the **manual VPS monitor GUI QA (`docs/VPS_MONITOR_QA.md`) was NOT executed** — treat live panel behavior as caveated until user-tested. See "v1.2.0 Changes to Be Aware Of". **Do not mark any VPS_MONITOR_QA case PASS without a real human run.**

**Latest release: v1.1.0** (2026-07-05 — tag `v1.1.0`, GitHub Latest) — **SFTP Two-Pane Foundation**, released **with a documented GUI-QA caveat**. It adds a read-only local pane (`internal/localfs`: List/Home/Roots/Parent/Exists) beside the existing remote pane, recursive remote→local download (`sftpx.DownloadPaths`), and two-pane upload/download wiring with overwrite confirmation (`SftpExists`/`LocalExists`). Frontend: `SftpPane.vue` (local) + `SftpPanel.vue` (container, remote logic still inline); local cwd is in-memory only (never persisted). Its automated unit + build-tagged integration gate is green, but the **manual SFTP two-pane GUI QA (`docs/SFTP_TWO_PANE_QA.md`) was NOT executed** before release — the user chose to release with that caveat, so the GUI flows are caveated until user-tested. **Do not mark any SFTP_TWO_PANE_QA case PASS without a real human run.** Previous stable is **v1.0.0** (unchanged). Prior tags v0.4.0–v1.0.0 unchanged; **no separate v0.6.0 or v0.8.0 tag/Release**. Advanced SSH lives in `internal/hosts/advanced.go` (non-secret config) and `internal/sshsess/{manager,tunnel,socks,diag}.go`; secret scrubbing in `internal/redact`. ProxyJump is single-level; a manual bastion is key-only (password bastions must reference a saved host).

**After v1.0.0 the project is in a 1.x maintenance phase.** Do NOT auto-start a new feature track (no v1.1.0 features, no large refactors, no SFTP dual-pane rewrite, no editor/cloud-sync/plugin/account/telemetry/auto-updater). Future work should be small bugfix / patch releases (e.g. v1.0.1) and only on explicit direction. Every release must pass the full gate: `go test ./...` · `go vet ./...` · `go mod verify` · `go test -tags=integration ./...` · `npm run build` · `build-windows.bat`. Never commit `data/`, secrets, `qa-local/`, `build/`, `frontend/dist/assets`, release zips, or logs. **Open QA item:** GUI auto-reconnect UX (cap/cancel/discriminator) has not been separately human-tested — the backend close signal is covered by the integration tests only. A ready-to-run manual checklist exists at `docs/GUI_AUTO_RECONNECT_QA.md` (authored 2026-07-04, not yet human-executed; no case marked PASS). Execute it on a throwaway host before relying on the reconnect UX.

---

## How Wails Works (the most likely source of confusion)

Wails wraps a Go binary with a WebView2 window. The Go side exposes methods via `Bind: []interface{}{app}`
in `main.go`. These become callable as `window.go.main.App.MethodName(args)` in JavaScript.

- **Go → JS**: `runtime.EventsEmit(ctx, "event:name", payload)` in Go; `window.runtime.EventsOn("event:name", fn)` in JS.
- **JS → Go**: direct call `await window.go.main.App.SomeMethod(arg1, arg2)`.
- **Type bridge**: `frontend/src/wails.d.ts` is the TypeScript type contract for the Go API. If you add a Go method, add its type here too. If you add/change a struct field returned from Go, update the matching type in `wails.d.ts`.
- **Bindings regeneration**: `wails dev` or `wails build` auto-regenerates `frontend/wailsjs/`. Never edit those files manually.
- **File drag-and-drop**: requires `DragAndDrop: &options.DragAndDrop{EnableFileDrop: true}` in `main.go` AND `runtime.OnFileDrop(ctx, callback)` in `startup()`. Both must be present or drop is silently broken.

---

## Data Layout (`internal/portable`)

Everything is anchored to the executable's directory. Never use `os.UserHomeDir()` or hardcode paths.

```
<exe-dir>/
└── data/
    ├── secret.key       32-byte AES key, generated once, 0600 perms
    ├── settings.json    user preferences
    ├── hosts.json       saved connections (passwords AES-encrypted)
    ├── known_hosts      SSH host fingerprints
    └── keys/
        ├── index.json   keypair metadata
        ├── <id>.pub     plaintext public key
        └── <id>.key.enc AES-256-GCM encrypted private key
```

`internal/portable` resolves all these paths. Always call `portable.DataPath("filename")` — never
construct paths manually.

---

## Encryption Model

`internal/cryptox` handles all encryption. Key points:
- Key is in `data/secret.key`. If it is lost, all stored passwords and passphrases are unrecoverable.
- Encryption: AES-256-GCM. Nonce is prepended (12 bytes), ciphertext follows, result is base64-encoded.
- The `cryptox` package is a singleton: key loaded (or generated) once on first call, then cached.
- Passwords stored in `hosts.json` as `encPassword` / `encPassphrase` fields. The `Host.Password` field in memory is always plaintext but never written to disk in that form.

---

## SSH Session Lifecycle (critical path)

1. Frontend calls `App.SshOpen(tabId, hostId, cols, rows)`.
2. `hosts.Get(hostId)` decrypts and returns credentials.
3. `sshsess.Manager.Open()` dials, verifies host key, starts PTY shell.
4. Two goroutines pump stdout and stderr → base64 → `EventsEmit("ssh:data:<tabId>")`.
5. A third goroutine waits for `sess.Wait()` → `EventsEmit("ssh:close:<tabId>")`.
6. `Terminal.vue` receives `ssh:data` events and writes to xterm.js.

**Do not add synchronous waits in the session goroutines.** They must remain non-blocking.

---

## What NOT to Change Without Careful Thought

| Item | Why |
|------|-----|
| `data/secret.key` handling | Changing the key format or location breaks all existing stored credentials |
| `hosts.json` field names | JSON field names are the serialization contract; renaming breaks existing files |
| `known_hosts` format | Must remain `ssh.knownhosts`-compatible; don't invent your own format |
| The module path `github.com/leungbzai-png/ssh-terminal` | Changing `go.mod` requires updating every import and breaks `go get` for existing users |
| `frontend/wailsjs/` contents | Auto-generated by Wails; never edit manually |
| `options.DragAndDrop{EnableFileDrop: true}` in `main.go` | Removing this silently breaks drag-drop upload |

---

## v0.3.0 Changes to Be Aware Of

**New `InputDialog.vue`** — generic text-input modal (props: `title`, `placeholder`, `defaultValue?`, `confirmLabel?`; emits: `confirm(value)`, `cancel`). Used by `SftpPanel.vue` for mkdir and rename.

**`sshsess.Manager.Open` signature changed** — now accepts `timeoutSec int` as last parameter. Call site is `app.go:OpenSession`. Value read from `config.Settings.ConnectTimeoutSec`.

**`sftpx.Manager.DeleteRecursive`** — wraps `sftp.Client.RemoveAll`. Guard: rejects `""`, `"/"`, `"."`. Called via `app.go:SftpDeleteRecursive`. Safe to use on files too (falls back to single `Remove`).

**CI is live** — `.github/workflows/ci.yml` runs on every push to `main`. Two jobs: Go (windows-latest), Frontend (ubuntu-latest). No `wailsjs/` dependency in the frontend — all Wails calls use `window.go.*` globals.

---

## v0.4.0 Changes to Be Aware Of

**`sshsess.Manager.Open` signature changed again** — now `Open(sessionID, h, cols, rows, timeoutSec, keepAliveSec int)`. `keepAliveSec > 0` starts a keepalive goroutine; 0 disables it. `app.go:keepAliveSecFrom(settings)` computes the effective value (0 when disabled, else the interval with a 30 s fallback). Both `OpenSession` and `SshOpenQuick` call `Open`.

**KeepAlive goroutine lifecycle** — `Session` now has a `done chan struct{}` initialized in `Open` and `close(done)`d exactly once inside the `s.closed`-guarded block of `closeWithReason`. `startKeepAlive` selects on `done` + a ticker and only ever calls `client.SendRequest("keepalive@openssh.com", ...)`; it never touches stdin/stdout/stderr, so it cannot block the io pumps or the session-wait goroutine.

**`SshOpenQuick(sessionID, QuickConnectParams, cols, rows)`** — opens a session from ephemeral credentials by building an **in-memory** `hosts.Host` and calling `Open`. It must NEVER persist. The plaintext password/passphrase exist only for the request; on the frontend they live in `sessions.quickParams[tabId]` and are deleted in `closeTab`. "Remember this host" is a separate frontend `UpsertHost` call (the normal encrypted path) — not part of `SshOpenQuick`.

**`internal/sshconfig` package** — pure OpenSSH config parser. `Parse(io.Reader) []Entry` does NO filesystem access (keywords case-insensitive, `Key=Value` and `Key Value` both accepted, multi-pattern `Host` uses first concrete alias, wildcard-only blocks like `Host *` are skipped and their directives never leak forward). `~` expansion is `ExpandUser` (separate, uses `os.UserHomeDir`). `DefaultPath()` returns `~/.ssh/config`. Unit tests live in `sshconfig_test.go`.

**Import APIs** (`app.go`): `DefaultSshConfigPath()`, `PickSshConfig()`, `PreviewSshConfig(path)` → `[]SshConfigPreviewEntry` (adds `identityExists`/`duplicate`), `ImportSshConfig(entries)` → `SshConfigImportResult`. Imported `IdentityFile` hosts use `authType:"key"` with `KeyPath` pointing at the **external** file — no key is copied into `data/`, nothing is decrypted. Duplicates (address+port+user, case-insensitive) are skipped.

**Adding a new auth type still means updating BOTH** `buildAuth` and `buildAuthForDeploy` (unchanged from v0.3.0), plus `QuickConnectDialog.vue`'s auth `<select>` if it should be quick-connectable.

## v0.5.0 Changes to Be Aware Of

**⚠ No-plaintext-secrets policy (read before touching hosts/keys/export).** These must NEVER be written to disk in plaintext or leave the app in an export: SSH password, private key, key passphrase, Quick Connect temporary secrets, imported key material, API tokens. Allowed in plaintext: alias, hostname, port, user, auth type, group, note, public key, fingerprint, known_hosts, UI settings, key *references* (external paths). If you add a field, decide which list it belongs to and add a test.

**Safe host export/import (`internal/hosts/export.go`).** Export is built from a dedicated whitelist struct `hosts.SafeHost` — *not* `Host`/`storedHost` — so a secret field cannot be added by accident. `BuildExport()`/`MarshalExport()` produce the document; `ParseExport()` validates `format == "ssh-terminal.hosts.safe-export"` and `version`. The document carries only: name, address, port, user, authType, keyPath, managedKeyId, group, note. App APIs: `ExportHosts()` (SaveFileDialog → writes file → returns path), `PreviewHostsImport()` (OpenFileDialog → parse → annotate with `duplicate`/`keyExists`, no mutation), `ImportHosts(entries, overwrite)`. Import dedups by address+port+user via `findDuplicateHostID`; duplicates are skipped unless `overwrite` is true; new hosts get fresh IDs (incoming IDs are ignored — `Host.ID` is left empty so `Upsert` mints one). Overwrite updates in place and preserves the existing encrypted password.

**Encrypted private-key import (`internal/keymgr/import.go`).** `ImportFromFile(name, comment, path, passphrase)` reads the key file on the Go side (plaintext never crosses the Wails bridge, never logged), validates with `ssh.ParsePrivateKey` (on `*ssh.PassphraseMissingError`, requires the passphrase and uses `ParsePrivateKeyWithPassphrase`), then cryptox-encrypts the **original** bytes to `data/keys/<id>.key.enc`. It never strips the passphrase and never persists it. `HasPassword` reflects the key's real protection. App API: `ImportPrivateKey(name, comment, keyPath, passphrase)`. Only `.key.enc`, `<id>.pub`, and `index.json` metadata are written under `data/keys` — no `.pem`/`.key`/`id_rsa` plaintext ever.

**Host groups + search (frontend, `Sidebar.vue`).** Grouping key is `h.group?.trim() || "Ungrouped"`; the `Ungrouped` virtual group always sorts last. Search filters by name/address/user/group (case-insensitive) and hides empty groups. `HostDialog.vue` group input uses a `<datalist>` of existing groups. No backend change was needed (the `group` field already existed).

**Security tests.** `internal/hosts/export_test.go` and `internal/keymgr/import_test.go`. The automated scan targets ONLY generated temp-dir artifacts (hosts.json, export bytes, files under `data/keys`) and asserts on unambiguous markers: a unique per-test sentinel secret value (cannot false-positive) plus PEM headers (`PRIVATE KEY`). It deliberately does NOT grep the repo/source/docs — those legitimately contain PEM strings (in policy docs) and key-path substrings like `id_ed25519`. The filename markers (`id_rsa`, `.pem`, `.key`) are for the *manual* QA scan, where a human can tell a path reference from key material.

## v0.6.0 / v0.7.0 Changes to Be Aware Of (Terminal + SFTP UX)

**Tab restore (`internal/session`).** Persists only non-secret tab intent (`{hostId, hostName}`) to `data/session.json`. App APIs `GetOpenTabs()` / `SaveOpenTabs()`. The frontend `sessions` store adds `status:"idle"` and `openSavedTabIdle(host)`, and debounces `SaveOpenTabs` on every open/close (only tabs with a non-empty `hostId` and `!quick` are written — Quick Connect secrets never persist). `Terminal.vue` skips `startSession()` when status is `"idle"` and shows a "Ready to connect" overlay whose button calls `startSession`. `App.vue:restoreTabs()` runs after hosts load and **skips hosts that no longer exist** (no crash). Restore never auto-connects.

**SFTP transfer progress — dedicated event namespace.** New tracked transfers emit on `sftp:xfer:progress:<tabId>` / `sftp:xfer:done:<tabId>` (payload includes `direction`). This is deliberately separate from the drag-upload events (`sftp:progress`/`sftp:done`) used by `App.vue`, because Wails `EventsOff(name)` removes *all* listeners for a name — sharing them would let App.vue's transient drag `onDone` clobber SftpPanel's persistent listeners. `SftpPanel` owns the xfer listeners (registered in onMounted, `EventsOff` in onBeforeUnmount). Backend: `sftpx.DownloadWithProgress`, and `app.go` `SftpDownloadTracked`/`SftpUploadTracked` + the `xferEmitters` helper. The old `SftpUploadPaths` (drag) is unchanged.

**Remote bookmarks (`internal/bookmarks`).** Per-host `{id, hostId, name, path, createdAt}` in `data/bookmarks.json`. Non-secret; **not** part of the safe host export (a separate file). App APIs `ListBookmarks(hostId)` / `AddBookmark(hostId,name,path)` (idempotent on host+path) / `DeleteBookmark(id)`. `List("")` returns nothing (bookmarks are host-scoped). Quick Connect tabs have no `hostId` → the SFTP bookmark menu shows a not-supported hint.

**Text preview.** `sftpx.IsProbablyText` (pure: NUL byte or invalid UTF-8 ⇒ binary) + `sftpx.ReadFilePreview` (size-capped, rejects directories). `app.go SftpPreviewText` caps at 512 KiB, returns `{content, size, tooLarge, binary}` — read-only, never writes the remote file. Frontend `TextPreviewDialog.vue`; opened by double-clicking a text-extension file or right-click → 预览.

**Font size shortcuts.** Handled in `Terminal.vue:onTermKey` (Ctrl+= / Ctrl+- / Ctrl+0) → `settings` store `bumpFontSize`/`resetFontSize` (clamped 8–32, persisted). `applySettings` already reacts to the deep settings watch, so all terminals update.

**New persisted files are non-secret.** `data/session.json` and `data/bookmarks.json` join `settings.json` as plaintext-allowed (host references / UI state only). Tests assert they contain no secret/PEM markers. If you add a field, put it on the allowed or forbidden list and add a test.

## v1.2.0 Changes to Be Aware Of (VPS Monitor Sidebar)

**`internal/sysmon` — pure parser package (no SSH, no I/O).** Parsers for
`/proc/stat` (`ParseStat` → `CPUCounters{Total,Idle}`; Total sums all cpu fields,
Idle = idle+iowait), `/proc/meminfo` (`ParseMeminfo`; MemTotal/MemAvailable
**required** → error if missing, swap optional), `df -P /` (`ParseDf`; uses the
last data line and anchors on the `NN%` Capacity column so spaced filesystem
names still parse), `/proc/loadavg`, `/proc/uptime`, and `uname -s`
(`ParseUname` → supported only when `== "Linux"`). `ParseAll(raw)` splits the
combined output on `@@OS@@/@@STAT@@/@@MEM@@/@@LOAD@@/@@UP@@/@@DF@@` markers and
**degrades gracefully** (non-Linux returns `Supported=false` early; missing
sections stay zero; never panics). It deliberately leaves `CPUPercent=0 /
CPUValid=false` — CPU needs a delta. `StatCounters(raw)` extracts the counters to
feed the delta. `Manager` (per-session `map[sessionID]CPUCounters`) computes
`100*(1 - idleDelta/totalDelta)` in `Sample`; first sample / zero-delta / counter
reset all return `valid=false` while updating the baseline. Fully unit-tested
(`sysmon_test.go`), no runtime dependencies.

**`sshsess.Manager.Run(sessionID, cmd)` — one-off exec on a SEPARATE channel.**
Grabs the session's `*ssh.Client` and opens a NEW channel via `client.NewSession()`
+ `CombinedOutput` (mirrors `DeployPublicKeyToHost`/SFTP). It never touches the
interactive shell PTY, so monitoring cannot disturb the terminal, and it runs
concurrently over the same multiplexed connection. A non-zero remote exit still
returns usable stdout (only a missing session / transport error is fatal). Add
new one-off remote commands here, not through the shell stdin.

**`App.MonitorSample(sessionID)` bridge + `sysmon.Manager` on `App`.** Runs the
fixed `sysmon.Command` (a constant string — **no session/user interpolation, no
injection surface**), parses with `ParseAll`, and for Linux hosts computes the
CPU delta via the `App.sysmon` manager. **Disconnected → error** (UI shows
no-session); **non-Linux → successful `MonitorSnapshot{Supported:false}`** (UI
shows unsupported) — keep these two distinct. `CloseSession` calls
`a.sysmon.Forget(sessionID)` alongside `a.sftp.Close`. Build-tagged integration
test `TestIntegrationManagerRun` (exec handler added to the in-process test
server) covers the round-trip and `Run → ParseAll`.

**Frontend: `MonitorSidebar.vue` owns the poll timer; state lives in the store.**
The sidebar is the left grid column in `PaneView` (`.split` uses
`data-monitor`/`data-sftp` for all on/off combos; toggling it resizes the terminal
element, whose existing `ResizeObserver` refits xterm). A single `setInterval`
calls `MonitorSample` on the per-tab interval (2/5/10s); an `inFlight` guard
prevents overlap and the tab id / connection are re-checked after the `await`.
The panel instance is **reused across tab switches** (like `SftpPanel`), so
per-tab monitor state (interval / snapshot / error / CPU+mem trend, capped at 40)
lives in the `sessions` store keyed by tab id and is cleared in `closeTab`;
`showMonitor` doubles as the per-tab enable/disable. **All monitor data is
in-memory only — never persisted** (no new `data/` file). Timer clears on unmount
/ tab-change / interval-change; disconnect resets the tab's reading. Only the
active tab's sidebar is mounted, so there is no background polling.

## Known Gotchas

**1. `buildAuthForDeploy` vs `buildAuth`**  
Two nearly identical auth switch blocks: `app.go:buildAuthForDeploy` and `internal/sshsess/manager.go:buildAuth`. If you add a new auth type, update BOTH. A TODO comment marks the duplicate in `app.go`.

**2. `sshsess.appendKnownHost` writes two entries**  
Both the hostname and the `host:port` form are written. This is intentional but means `known_hosts` may have two lines per host. Don't "deduplicate" it without understanding why both forms exist.

**3. `frontend/dist/.gitkeep` is deleted on every build**  
`vite build` clears the output directory. `build-windows.bat` recreates `.gitkeep` after the build. If you write a new build script or use `wails build` directly, recreate it: `type nul > frontend\dist\.gitkeep`.

**4. `useTheme.ts` is a module-level singleton**  
`mode`, `resolved`, and the `matchMedia` listener all live at module scope. The `useTheme()` composable just exposes the shared refs. Do not add lifecycle hooks (`onMounted`/`onUnmounted`) inside `useTheme()` — they were removed in v0.2.0 to fix a listener leak.

**5. Wails events must be unregistered on component unmount**  
Pattern in `App.vue`: `EventsOn("app:filedrop", fn)` in `onMounted`, `EventsOff("app:filedrop")` in `onBeforeUnmount`. Always pair them.

**6. Tab IDs are generated in the frontend**  
`uid()` in `stores/sessions.ts` uses `Date.now().toString(36)` + counter. Session IDs used as event keys (e.g., `ssh:data:<tabId>`). If you change the generation scheme, ensure uniqueness under rapid tab creation.

---

## Adding a New Exposed Go Method

1. Add the method to `app.go` with correct signature.
2. Run `wails build` (or `wails generate`) — this regenerates `frontend/wailsjs/go/main/App.js` and `App.d.ts`.
3. Add the TypeScript type to `frontend/src/wails.d.ts` manually (Wails generates runtime bindings but not the hand-maintained type file).
4. Call it in Vue via `window.go.main.App.YourMethod(arg)`.

---

## Adding a New Host Auth Type

If you add a new `authType` value:
1. Update `app.go:buildAuthForDeploy` switch.
2. Update `internal/sshsess/manager.go:buildAuth` switch (same logic, keep in sync).
3. Update `frontend/src/components/HostDialog.vue` `<select>` options.
4. Update `frontend/src/wails.d.ts` `HostRecord.authType` type union.
5. Test: new host with new auth type connects successfully.

---

## Performance Notes

- Terminal output is Base64-encoded for the Wails event bridge. This is ~33% overhead but negligible at SSH speeds.
- `Terminal.vue` uses `v-show` (not `v-if`) for tabs so xterm.js instances persist in memory; this is intentional for scroll-history preservation.
- xterm.js `FitAddon` is called on `ResizeObserver` callback. Don't call it synchronously during render.

---

## Testing (Current State)

The project now has **unit tests** across `internal/*` (hosts, keymgr, redact,
sshsess diag/socks, session, bookmarks, sftpx, sshconfig) plus `app_test.go`.
Run them the normal way — fast, offline, no build tag:

```bash
go test ./...
```

**Advanced SSH integration suite (build-tagged, backend-live).** `internal/sshsess`
carries `integration_test.go` + `integration_server_test.go` behind
`//go:build integration`. They drive the real `Manager` against **disposable
in-process SSH servers on 127.0.0.1** — ProxyJump, local/remote/dynamic-SOCKS
forwarding, occupied-port resilience, connection diagnostics, runtime cleanup,
and the auto-reconnect close signal. They are **excluded** from `go test ./...`
and never touch a real server or secret (all credentials runtime-generated,
errors redacted). Run on demand:

```bash
go test -tags=integration ./...
go test -tags=integration ./internal/sshsess -run Integration -v
```

See `docs/INTEGRATION_TESTS.md` for coverage and limitations (GUI auto-reconnect
still needs manual QA; `-race` needs CGO/GCC on Windows). Use this suite as a
v1.0.0 readiness gate.
