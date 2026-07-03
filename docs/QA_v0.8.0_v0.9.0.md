# Manual QA Checklist — v0.8.0 + v0.9.0 (released as v0.9.0)

Combined Advanced SSH (v0.8.0) + Hardening (v0.9.0) scope. Run against the
Windows portable build (`build\bin\ssh-terminal.exe`). Data files are created in
a `data/` folder next to the exe on first run.

> **Runtime-verification note.** The automated gate (`go test`, `go vet`,
> `npm run build`, `build-windows.bat`) proves the code compiles and that the
> pure logic (SOCKS5 parsing, error classification, redaction, config
> validation, storage compat) is unit-tested. It does **not** exercise a live
> SSH server or bastion. The tunnel/ProxyJump/reconnect items below require a
> real SSH server (and, for ProxyJump, a bastion) and must be checked here.

Legend: ☐ not run · ✅ pass · ❌ fail

---

## A. ProxyJump / Bastion

- ☐ A1. Save a host with ProxyJump → **saved jump host**; connect succeeds through the bastion.
- ☐ A2. Save a host with ProxyJump → **manual** (address/user/**key path**); connect succeeds.
- ☐ A3. Manual ProxyJump with **no key path** is rejected at save (key-only enforced; no password field exists).
- ☐ A4. Jump-host authentication does **not** copy or persist the bastion's secret (inspect `hosts.json` — target host stores no jump secret).
- ☐ A5. Delete the referenced jump host, then connect the target → clear "跳板机主机缺失" error, **no crash**.
- ☐ A6. Quick Connect uses no jump config and persists no jump secret.
- ☐ A7. Safe export of a ProxyJump host contains the `proxyJump` block but **no** password/passphrase/private key/`.key.enc`/`secret.key`.
- ☐ A8. A v0.7.0 `hosts.json` (no `advanced` field) still loads and connects normally.

## B. Local Port Forwarding

- ☐ B1. Add a local forward (127.0.0.1:LP → RH:RP), connect → terminal shows "隧道已建立 [local/...]"; the local port serves the remote target.
- ☐ B2. Occupied local port → "隧道失败 … bind" message; session still opens (no crash).
- ☐ B3. Disconnect → the local port is released (re-run `netstat`/connect to confirm).
- ☐ B4. Close the tab → the local port is released.
- ☐ B5. Empty localHost defaults to 127.0.0.1.
- ☐ B6. Duplicate local bind (two enabled forwards on the same host:port, incl. dynamic) is rejected at save.

## C. Remote Port Forwarding

- ☐ C1. Add a remote forward (RH:RP → 127.0.0.1:LP), connect → tunnel-established notice.
- ☐ C2. Server refuses the bind (GatewayPorts) → readable "隧道失败 … server may reject" message, **no crash**.
- ☐ C3. Disconnect / close tab → tunnel cleaned up.
- ☐ C4. UI shows the GatewayPorts dependency note.

## D. Dynamic SOCKS5

- ☐ D1. Add a dynamic forward (127.0.0.1:1080), connect → SOCKS listener starts (notice shown).
- ☐ D2. Point a browser/curl at the SOCKS5 proxy → traffic tunnels through the host (IPv4 + domain targets).
- ☐ D3. Occupied port → failure message, no crash.
- ☐ D4. Disconnect / close tab → listener released.
- ☐ D5. Default bind is 127.0.0.1.

## E. Auto Reconnect

- ☐ E1. Enable auto-reconnect; kill the connection server-side (or drop the network) → app reconnects automatically; banner shows attempt count.
- ☐ E2. Manually close the tab → **no** auto-reconnect.
- ☐ E3. Type `exit` on the remote (clean exit) → **no** auto-reconnect.
- ☐ E4. Wrong credentials → reconnect stops at maxAttempts (does not loop forever).
- ☐ E5. Reach maxAttempts → reconnect stops with a "已达上限" notice.
- ☐ E6. Click **取消** during a pending reconnect → burst stops.
- ☐ E7. A successful reconnect resets the attempt counter (a later drop starts a fresh burst).

## F. Connection Diagnostics

- ☐ F1. Unknown hostname → "无法解析主机名（DNS 失败）".
- ☐ F2. Closed port / unreachable → "连接被拒绝" or "连接超时".
- ☐ F3. Wrong password/key → "认证失败".
- ☐ F4. Wrong key passphrase → "私钥或口令无效".
- ☐ F5. Bastion unreachable → "无法通过跳板机连接".
- ☐ F6. Port-forward bind failure → forward-specific message.
- ☐ F7. No error message contains a password, passphrase, or private key.

## G. Hardening

- ☐ G1. v0.7.0 data compatibility (also A8).
- ☐ G2. Manually corrupt `hosts.json` → app reports an error and does **not** panic or overwrite it with `[]`.
- ☐ G3. Invalid Advanced SSH input (bad port, missing remote host) is rejected at save with a clear message.
- ☐ G4. Safe export excludes secrets (also A7).
- ☐ G5. Session restore contains no secret (inspect `session.json`).
- ☐ G6. Bookmarks contain no secret (inspect `bookmarks.json`).
- ☐ G7. Tunnel config in `hosts.json` contains no secret.
- ☐ G8. Multiple tabs with tunnels do not cross-contaminate state; each tab's ports are independent.
- ☐ G9. App exit closes active sessions and tunnels (ports released).

## H. Regression (v0.7.0 scope still works)

- ☐ H1. Terminal search (Ctrl+F) with match count + no-match indicator.
- ☐ H2. Font family / size settings + Ctrl +/-/0.
- ☐ H3. Saved-host tab restore on launch (idle, no auto-connect).
- ☐ H4. Shortcut help dialog (F1).
- ☐ H5. SFTP upload/download progress, drag-upload overlay, bookmarks, text preview.
- ☐ H6. Host management (groups, search, create/edit/delete).
- ☐ H7. Secure storage: password/passphrase encrypted at rest; managed key generate/import (`.key.enc`); deploy public key.
- ☐ H8. Safe export/import round-trip.
- ☐ H9. Quick Connect (ephemeral, not persisted) + "记住此主机" (encrypted save).

## I. Build / Artifact

- ✅ I1. `go test ./...` passes.
- ✅ I2. `go vet ./...` passes.
- ✅ I3. `go mod verify` passes.
- ☐ I4. `go test -race ./...` — **not run** in this environment (no gcc/CGO toolchain; documented Windows limitation).
- ✅ I5. `npm run build` (vue-tsc + vite) passes.
- ✅ I6. `build-windows.bat` passes.
- ✅ I7. Release zip contains only `ssh-terminal.exe`, `README.md`, `LICENSE`.
