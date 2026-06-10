# Project Context — SSH Terminal

This document records **why** key decisions were made. Answers the question:
"Why is it built this way?" so future maintainers don't undo intentional choices.

---

## Why Go + Wails, Not Electron/Tauri?

**Electron** was ruled out: 100+ MB distributable, Chrome runtime overhead, Node.js security surface.

**Tauri** was considered: Rust backend is excellent but adds a second language. For a single developer maintaining a personal tool, Go's standard library + SSH ecosystem (`golang.org/x/crypto`) is more familiar and more directly capable.

**Wails v2** was chosen because:
- Go backend handles all SSH/SFTP natively using proven libraries
- WebView2 on Windows is already installed on Win11, tiny footprint
- Vue 3 frontend is fast to iterate and looks good
- The Go↔JS bridge is ergonomic (exported methods auto-bound)
- Resulting exe is ~11 MB stripped vs ~120 MB for Electron equivalent

The tradeoff: Windows-only (WebView2). macOS/Linux possible with Wails but not yet prioritized.

---

## Why Portable (Data Next to Exe)?

Design goal: zero footprint. The app should be "copy and run" — no installer, no registry writes, no `%APPDATA%` folders that survive uninstalls.

`internal/portable` resolves all paths relative to the executable's location. This enables:
- Running from a USB drive
- Multiple isolated instances (copy the whole folder)
- Clean uninstall (delete the folder)

The tradeoff: if you move only the exe without `data/`, all settings and credentials are lost. This is documented in README and is considered an acceptable trade-off for the target user.

---

## Why AES-256-GCM with a Local Key File?

The security model is explicitly "protect against casual disk inspection" — not against an attacker who has file system access to the machine.

Rationale:
- Passwords in plain `hosts.json` would be readable by any process with file system access.
- AES-256-GCM with a local key (`data/secret.key`) means credentials can't be read without also having the key.
- The key is machine-local and never transmitted. If you backup `data/` to the cloud without encrypting it, the key goes with it (risk accepted by the user).
- Using OS credential stores (Windows Credential Manager, etc.) would break portability. That trade-off was explicitly rejected.

**Recommendation to users:** Use SSH key authentication, not stored passwords. The key storage is a convenience fallback, not the primary security model.

---

## Why `golang.org/x/crypto/ssh` Directly, Not `libssh2`?

`golang.org/x/crypto/ssh` is the standard Go SSH library, maintained by the Go team, pure Go (no CGo), easy to cross-compile, and has excellent `knownhosts` support. It handles everything this app needs: PTY, session multiplexing, public key auth, password auth.

`libssh2` (used by some C/C++ tools) would require CGo and complicate the Windows build pipeline.

---

## Why Strict `known_hosts` Verification?

The app does not have a "Trust All Hosts" option. This is intentional.

Accepting all host keys silently is a MITM risk. Prompting on first connect and hard-failing on key mismatch is the standard SSH behavior. The app follows it.

The first-connect dialog shows the SHA-256 fingerprint. The user is expected to verify it out-of-band (e.g., via their server provider's console). This is documented in README.

---

## Why xterm.js Instead of a Native Terminal?

xterm.js is the de-facto standard browser-based terminal emulator, used in VS Code, Tabby, JupyterLab, etc. It has:
- Excellent VT100/VT220/xterm compatibility
- FitAddon for automatic resizing
- SearchAddon for Ctrl+F search
- WebLinksAddon for clickable URLs
- Active maintenance

Building a native terminal (Win32 ConPTY or similar) would be far more work and would reduce the portability of the frontend code.

The tradeoff: terminal rendering goes through WebView2 → xterm.js → canvas, so there is slight overhead vs a native terminal. In practice, this is not measurable for typical SSH usage.

---

## Why Vue 3 + Pinia, Not React?

Personal preference of the original developer. Vue 3 with `<script setup>` and Pinia is ergonomic for small-to-medium single-page apps. The Composition API makes it easy to share state (like `useTheme`) without prop drilling.

No framework migration is planned.

---

## Why `v-show` Instead of `v-if` for Terminals?

`Terminal.vue` uses `v-show` (CSS `display:none`) rather than `v-if` (DOM removal) to hide inactive tabs. This preserves the xterm.js instance and its scroll history in memory.

If `v-if` were used, the terminal would be destroyed and recreated on tab switch, losing scroll history and requiring a re-render. The memory cost (~5–10 MB per terminal instance) is acceptable.

---

## Why Base64 for Terminal Data?

The Wails event bridge serializes payload as JSON. Binary data (SSH output bytes) cannot be transmitted as raw bytes through JSON. Base64 encoding converts binary to a JSON-safe string.

The overhead (~33% size increase) is negligible at SSH bandwidth speeds (typically <1 MB/s for interactive sessions).

---

## Why Module-Level Singleton for `useTheme`?

`theme`, `resolved`, and the `matchMedia` listener are all module-level. The `useTheme()` composable just exposes the shared refs without registering lifecycle hooks.

Alternative (component-scoped state with lifecycle hooks) was the original implementation but caused `matchMedia` listener leaks when multiple components called `useTheme()`. The module-level approach is correct for truly global state with no teardown needed.

---

## Known Limitations Accepted as Design Decisions

| Limitation | Reason Accepted |
|-----------|-----------------|
| Windows only | WebView2 is Windows-specific; cross-platform not a current goal |
| No recursive SFTP delete | Preventing accidental mass deletion; requires explicit implementation |
| No ProxyJump | Not yet implemented; planned for v0.4.0 |
| `data/secret.key` is irrecoverable if lost | By design; forcing key backup is not in scope for a personal tool |
| `known_hosts` may have duplicate entries | Cosmetic; does not affect security or functionality |
| `buildAuth` duplicated in two places | Minor tech debt; acceptable until a third caller appears |
