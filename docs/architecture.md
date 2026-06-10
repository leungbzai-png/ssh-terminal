# Architecture Reference

This document describes the internal structure of SSH Terminal for maintainers.
All content is derived from the actual source code.

---

## High-Level Layout

```
main.go          Wails entry point — window config, DragAndDrop options
app.go           Go↔JS bridge: all methods exposed to the frontend via Bind
internal/
  portable/      Path resolution relative to the exe (DataDir, KeysDir, etc.)
  cryptox/       AES-256-GCM encryption; manages secret.key on disk
  config/        Reads/writes settings.json (theme, font, cursor, etc.)
  hosts/         Reads/writes hosts.json; encrypts passwords at rest
  keymgr/        SSH keypair generation (Ed25519/RSA), encrypted storage, index
  sshsess/       SSH client sessions with PTY, known_hosts verification
  sftpx/         SFTP file operations, batch upload with progress callbacks
frontend/src/
  App.vue        Root layout; drag-and-drop handler; app-level Wails events
  components/    UI components (see below)
  stores/        Pinia stores: settings, hosts, sessions
  composables/   useTheme (module-level singleton)
  style.css      CSS design tokens + light/dark themes
  wails.d.ts     TypeScript types for the entire Go API
```

---

## Data Flow

### SSH Session Lifecycle

```
Frontend: sessions.openInActivePane(host)
  → window.go.main.App.SshOpen(tabId, hostId, cols, rows)
    → hosts.Get(hostId)           decrypt credentials from hosts.json
    → sshsess.Manager.Open()      dial TCP, verify host key, start PTY shell
      → go pump(stdout)           goroutine: read → base64 → EventsEmit "ssh:data:<id>"
      → go pump(stderr)           goroutine: same
      → go sess.Wait()            goroutine: on exit → closeWithReason → EventsEmit "ssh:close:<id>"

Frontend Terminal.vue:
  EventsOn("ssh:data:<id>")  → base64 decode → terminal.write()
  EventsOn("ssh:close:<id>") → show reconnect overlay
```

### Host Key Verification (first connect)

```
sshsess.hostKeyCallback():
  knownhosts.New(data/known_hosts) → verify()
  if unknown: Manager.prompt(hostname, fp)
    → EventsEmit "ssh:hostkey" {hostname, fingerprint}
    → blocks on channel <ch>

Frontend HostKeyDialog.vue:
  EventsOn("ssh:hostkey") → show dialog
  user accept: App.AnswerHostKey(fp, true)
    → ch <- true
    → appendKnownHost writes to data/known_hosts
  user reject: App.AnswerHostKey(fp, false) → connection fails
```

### File Drag-and-Drop Upload

```
OS drops files on app window
  → Wails DragAndDrop (EnableFileDrop: true)
  → runtime.OnFileDrop callback → App.onFileDrop(x, y, paths)
    → EventsEmit "app:filedrop" {paths}

Frontend App.vue:
  EventsOn("app:filedrop") → onFileDrop()
    → sessions.sftpCwd[tabId] or App.SftpCwd(tabId)
    → user confirm dialog
    → EventsOn "sftp:progress:<tabId>", "sftp:done:<tabId>"
    → App.SftpUploadPaths(tabId, paths, remoteDir)
      → sftpx.Manager.UploadPaths()
        phase 1: walk paths, sum total bytes
        phase 2: upload files one by one, emit sftp:progress per chunk
        → EventsEmit "sftp:done:<tabId>" {ok, err}
```

### Credential Encryption

```
Encrypt(plaintext):
  cryptox.key (32-byte, loaded from data/secret.key or generated once)
  → AES-256-GCM
  → nonce (12 random bytes) + ciphertext
  → base64 encode
  → stored in hosts.json as encPassword / encPassphrase

Decrypt(ciphertext):
  base64 decode → split nonce | ciphertext → GCM.Open
```

---

## Internal Package Contracts

### `internal/portable`
Single exported function per path type. All return absolute paths under `<exe-dir>/data/`.
- `DataDir()` — `data/`
- `DataPath(name)` — `data/<name>`
- `KeysDir()` — `data/keys/`

Uses `sync.Once`; safe to call from any goroutine at any time.

### `internal/cryptox`
- Key is loaded (or generated) on first call; subsequent calls use the cached key.
- No external state beyond `data/secret.key`.
- Losing `secret.key` = losing all stored passwords and passphrases (unrecoverable by design).

### `internal/hosts`
- Package-level `cache map[string]storedHost` loaded once via `ensureLoaded()`.
- `List()` — strips secrets (safe for frontend).
- `Get(id)` — decrypts and returns secrets (used only by SSH/deploy code).
- `Upsert()` — preserves existing encrypted value if the incoming plaintext field is empty (allows UI to "not change" a stored password).
- Atomic write: `hosts.json.tmp` → rename.

### `internal/sshsess`
- `Manager` holds a map of active sessions keyed by tab ID.
- `Session.closeWithReason` uses a mutex to guarantee single-close.
- `buildAuth` — shared auth logic used by `Open`; a separate copy (`buildAuthForDeploy` in `app.go`) handles one-shot deploy connections.
- Known-hosts: `appendKnownHost` writes both normalized hostname and `host:port` to `data/known_hosts`. This can produce two entries per first-connect; harmless but worth knowing when editing the file manually.

### `internal/keymgr`
- Private keys: AES-256-GCM via cryptox → `data/keys/<id>.key.enc`
- Public keys: plaintext PEM → `data/keys/<id>.pub`
- Index: `data/keys/index.json`
- `LoadSigner(id, passphrase)` — decrypts `.key.enc`, parses PEM, returns `ssh.Signer`.
- RSA minimum: 2048 bits; default: 4096 bits.

### `internal/sftpx`
- Lazy SFTP client per session: created on first SFTP call, reused thereafter.
- `UploadPaths(tabId, localPaths, remoteDir)` — runs in a goroutine; progress is reported via Wails events `sftp:progress:<tabId>` and `sftp:done:<tabId>`.
- `Delete()` removes files and empty directories only (no recursive delete on directories).

---

## Frontend Architecture

### Pinia Stores

**`stores/sessions.ts`** — source of truth for all UI session state
- `panes: Pane[]` — max 4; each pane has its own tab bar
- `tabs: Record<tabId, Tab>` — flat map; tabs belong to exactly one pane
- `activePaneId` — which pane receives keyboard / new-tab actions
- `sftpCwd[tabId]` — last known remote directory per tab (cached to avoid round trips)
- `bumpReconnect(tabId)` / `bumpSftpRefresh(tabId)` — increment tick refs that Terminal.vue / SftpPanel.vue watch to trigger reconnects/reloads

**`stores/hosts.ts`** — mirrors `App.ListHosts()` result; refreshed on mount and after mutations

**`stores/settings.ts`** — persisted via `App.GetSettings()` / `App.SaveSettings()`

### Component Hierarchy

```
App.vue
├── Sidebar.vue            host list + buttons (new, settings, keys)
├── PaneView.vue × N       one per pane (max 4)
│   ├── TabBar.vue         tabs + right-click context menu
│   ├── Terminal.vue       xterm.js instance (one per tab, v-show not v-if)
│   ├── SftpPanel.vue      file browser (shown when tab.showSftp)
│   └── CommandBar.vue     broadcast command bar (shown per app settings)
├── HostDialog.vue         modal (v-if)
├── SettingsDialog.vue     modal (v-if)
├── KeysDialog.vue         modal (v-if)
├── HostKeyDialog.vue      modal (always mounted, hidden when no pending prompt)
└── CloseConfirmDialog.vue modal (v-if)
```

### Wails Event Bus (Go → JS)

| Event | Payload | Consumer |
|-------|---------|----------|
| `ssh:data:<tabId>` | `string` (base64) | `Terminal.vue` |
| `ssh:close:<tabId>` | `string` (reason) | `Terminal.vue` |
| `ssh:hostkey` | `{hostname, fingerprint}` | `HostKeyDialog.vue` |
| `app:confirmClose` | `number` (active count) | `App.vue` |
| `app:filedrop` | `{paths: string[]}` | `App.vue` |
| `sftp:progress:<tabId>` | `{transferred, total, current}` | `App.vue` |
| `sftp:done:<tabId>` | `{ok: bool, err: string}` | `App.vue` |

### CSS Design Tokens (`style.css`)

All UI colors and the xterm.js terminal palette are driven by CSS custom properties on `:root[data-theme="light"]` and `:root[data-theme="dark"]`. `Terminal.vue` reads these at mount time via `getComputedStyle(document.documentElement)` to pass into xterm.js `ITheme`. Changing a token automatically updates both the app chrome and the terminal colors.

---

## Known Limitations (as of v0.2.0)

- **Windows only**: Wails WebView2 backend; no macOS/Linux build pipeline.
- **No ProxyJump / SSH tunneling**: single-hop connections only.
- **SFTP Delete is non-recursive**: `sftpx.Delete()` only removes empty directories.
- **Known-hosts entries may be duplicated**: `appendKnownHost` writes both hostname and `host:port` forms on first connect.
- **`buildAuthForDeploy` duplicates `buildAuth`**: two copies of the auth switch exist (`app.go` and `internal/sshsess/manager.go`). Consolidate when a third caller appears.
