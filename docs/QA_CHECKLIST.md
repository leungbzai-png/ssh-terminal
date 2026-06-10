# QA Checklist — SSH Terminal

Version tested: ___________  
Tester: ___________  
Date: ___________  
Build: `build\bin\ssh-terminal.exe`

Legend: ✅ Pass  ❌ Fail  ⚠️ Partial / Workaround  — Not tested

---

## Known Issues (Carry-forward from v0.2.0)

These are documented defects, not regressions. Track them separately.

| ID | Severity | Description |
|----|----------|-------------|
| KI-01 | Medium | `SftpPanel` uses browser-native `prompt()`/`confirm()` dialogs for rename/mkdir/delete — inconsistent with app style |
| KI-02 | Medium | `SftpPanel.Delete()` cannot remove non-empty directories (SFTP limitation by design; no recursive delete) |
| KI-03 | Low | `data/known_hosts` may accumulate duplicate entries (hostname + IP) on first connect to same host |
| KI-04 | Low | TTY baud rate hardcoded at 14400 in `sshsess/manager.go` (ignored by modern SSH, cosmetic) |
| KI-05 | Low | UI language mixing: most UI is Chinese, a few sidebar labels are English |
| KI-06 | Low | 80 ms sleep before `runtime.Quit()` in `ConfirmQuit()` — not user-visible but inelegant |

---

## Pre-Release Checks

### Environment

| # | Check | Result |
|---|-------|--------|
| E-01 | App launches on fresh Windows machine (no `data/` directory present) | |
| E-02 | `data/` directory auto-created on first launch | |
| E-03 | `data/secret.key` created (32 bytes, 0600 equivalent) | |
| E-04 | App window opens at 1280×820, respects minimum 900×560 | |
| E-05 | Light / dark / system theme all apply correctly at startup | |
| E-06 | System theme tracks OS light/dark switch without restart | |
| E-07 | App exits cleanly when no sessions are open | |
| E-08 | Close-with-active-sessions confirmation dialog appears (setting enabled) | |
| E-09 | Portable: copy exe + `data/` to different directory, app still works | |
| E-10 | Portable: copy exe + `data/` to different machine, app still works | |

### Build Integrity

| # | Check | Result |
|---|-------|--------|
| B-01 | `go build ./...` passes with no errors | |
| B-02 | `go vet ./...` passes with no warnings | |
| B-03 | `wails build` completes without errors | |
| B-04 | Resulting exe file size is ~11 MB (stripped, trimpath) | |
| B-05 | `build\bin\data\` directory is NOT included in release zip | |
| B-06 | `build\bin\ssh-terminal.exe` is NOT committed to git | |

---

## SSH Tests

### Basic Connection

| # | Check | Result | Notes |
|---|-------|--------|-------|
| S-01 | Connect with **password** authentication | | |
| S-02 | Connect with **external key file** (no passphrase) | | |
| S-03 | Connect with **external key file** (with passphrase) | | |
| S-04 | Connect with **managed Ed25519 key** (no passphrase) | | |
| S-05 | Connect with **managed Ed25519 key** (with passphrase) | | |
| S-06 | Connect with **managed RSA key** | | |
| S-07 | Connect with wrong password → shows error, no crash | | |
| S-08 | Connect to unreachable host → shows timeout error (~15s) | | |
| S-09 | Connect to host with wrong port → shows error | | |

### Host Key Verification

| # | Check | Result | Notes |
|---|-------|--------|-------|
| S-10 | First connect to unknown host: fingerprint dialog appears with correct SHA-256 | | |
| S-11 | Accept fingerprint: host added to `data/known_hosts`, future connects skip dialog | | |
| S-12 | Reject fingerprint: connection fails cleanly | | |
| S-13 | Host key mismatch (edit `known_hosts` to wrong key): connect fails with "possible MITM" | | |
| S-14 | Edit `known_hosts` to remove host entry: fingerprint dialog reappears on next connect | | |

### Terminal Interaction

| # | Check | Result | Notes |
|---|-------|--------|-------|
| S-15 | Terminal output renders correctly (color, bold, underline) | | |
| S-16 | Terminal input works (type commands, see output) | | |
| S-17 | Ctrl+C sends interrupt signal | | |
| S-18 | Ctrl+D closes shell / logs out | | |
| S-19 | Arrow keys work (command history navigation in shell) | | |
| S-20 | Tab completion works | | |
| S-21 | Unicode output renders correctly (Chinese, emoji, box-drawing chars) | | |
| S-22 | Terminal resizes correctly when window is resized | | |
| S-23 | Terminal resizes correctly when SFTP panel is toggled | | |
| S-24 | Ctrl+F opens in-terminal search bar | | |
| S-25 | Search finds and highlights matches | | |
| S-26 | Scrollback works (default 5000 lines) | | |
| S-27 | `vim` / `nano` / full-screen TUI apps render correctly | | |

### Multi-Tab / Split Pane

| # | Check | Result | Notes |
|---|-------|--------|-------|
| S-28 | Open 2 tabs in same pane, both sessions independent | | |
| S-29 | Switch between tabs: terminal state preserved | | |
| S-30 | Close tab: session closed, no crash | | |
| S-31 | Right-click tab → Reconnect | | |
| S-32 | Right-click tab → Clone session: new tab opens same host | | |
| S-33 | Split right: second pane appears with independent tabs | | |
| S-34 | 3-pane and 4-pane layout: all function independently | | |
| S-35 | Close pane: remaining panes reflow correctly | | |

### Disconnect & Reconnect

| # | Check | Result | Notes |
|---|-------|--------|-------|
| S-36 | Remote shell exits (`exit` command): reconnect overlay appears | | |
| S-37 | Network interruption: reconnect overlay appears | | |
| S-38 | Click reconnect button: session re-established | | |

---

## SFTP Tests

| # | Check | Result | Notes |
|---|-------|--------|-------|
| F-01 | SFTP panel opens when clicking folder icon in tab bar | | |
| F-02 | SFTP panel shows current directory and file list | | |
| F-03 | Navigate into subdirectory (double-click) | | |
| F-04 | Navigate up one level (parent button or `..`) | | |
| F-05 | Breadcrumb / path display correct | | |
| F-06 | **Upload single file** via context menu or upload button | | |
| F-07 | **Upload multiple files** at once | | |
| F-08 | **Download single file** to local | | |
| F-09 | **Rename file** (via context menu) | | |
| F-10 | **Create directory** (via context menu) | | |
| F-11 | **Delete file** (single) | | |
| F-12 | Delete **empty directory** → success | | |
| F-13 | Delete **non-empty directory** → shows error (known KI-02) | | |
| F-14 | Upload progress displayed during large file transfer | | |
| F-15 | Unicode filenames upload/download correctly | | |
| F-16 | SFTP panel refreshes after file operations | | |
| F-17 | SFTP panel persists CWD when switching tabs and back | | |

---

## Drag-and-Drop Upload Tests

| # | Check | Result | Notes |
|---|-------|--------|-------|
| D-01 | Drag file from Explorer onto app: drop overlay appears | | |
| D-02 | Drop file: SFTP panel opens, upload begins, progress toast shows | | |
| D-03 | Drop directory: recursive upload completes, progress reflects total | | |
| D-04 | Drop multiple files: all uploaded, progress counts correctly | | |
| D-05 | Drop file with **no active session**: warning alert, no crash | | |
| D-06 | Drop file with session **not connected**: appropriate error | | |
| D-07 | Cancel drop (drag over then drag out): overlay disappears cleanly | | |
| D-08 | Upload completes: SFTP panel refreshes automatically | | |
| D-09 | Upload fails (no write permission on remote): error message shown | | |

---

## Key Management Tests

| # | Check | Result | Notes |
|---|-------|--------|-------|
| K-01 | Open Keys dialog (key icon in sidebar) | | |
| K-02 | Generate **Ed25519** key (no passphrase) | | |
| K-03 | Generate **Ed25519** key (with passphrase) | | |
| K-04 | Generate **RSA 2048** key | | |
| K-05 | Generate **RSA 4096** key | | |
| K-06 | Key appears in list with correct type and name | | |
| K-07 | Public key is viewable / copyable | | |
| K-08 | Delete key from list | | |
| K-09 | Deploy public key to host (via DeployKeyDialog) | | |
| K-10 | After deploy: connect with that key from HostDialog → success | | |
| K-11 | Select managed key with wrong passphrase → clear error | | |
| K-12 | `data/keys/` directory contains `.key.enc` and `.pub` files | | |
| K-13 | `data/keys/index.json` updated after generate/delete | | |

---

## Configuration Tests

| # | Check | Result | Notes |
|---|-------|--------|-------|
| C-01 | Open Settings dialog | | |
| C-02 | Change **theme** (light/dark/system): applies immediately | | |
| C-03 | Change **font family**: applies to open terminals | | |
| C-04 | Change **font size**: applies to open terminals | | |
| C-05 | Change **cursor style** (bar/block/underline) | | |
| C-06 | Toggle **cursor blink** | | |
| C-07 | Change **scrollback lines** | | |
| C-08 | Toggle **confirm close with active sessions** | | |
| C-09 | Toggle **show command bar** | | |
| C-10 | Settings persist after app restart | | |
| C-11 | Settings stored in `data/settings.json` (verify file content) | | |

---

## Command Bar Tests

| # | Check | Result | Notes |
|---|-------|--------|-------|
| CB-01 | Command bar visible (when enabled in settings) | | |
| CB-02 | Type command, press Enter: sent to **active tab only** | | |
| CB-03 | Type command, press Ctrl+Enter: sent to **all tabs in active pane** | | |
| CB-04 | Up/down arrow: navigates command history | | |
| CB-05 | History persists across sessions (stored in localStorage) | | |
| CB-06 | History capped at 200 entries | | |
| CB-07 | Unicode / special characters transmitted correctly | | |

---

## Regression Checklist (after each code change)

Before merging any PR or tagging a release, verify:

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes  
- [ ] App launches without errors
- [ ] Can connect to at least one SSH host
- [ ] SFTP panel opens and shows files
- [ ] Settings dialog opens and saves
- [ ] Theme switching works
