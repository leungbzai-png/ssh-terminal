# SFTP Two-Pane — Manual QA Checklist

**Status of this document:** *authored, not yet human-executed.* No case below is
marked PASS. Fill in Pass/Fail only after a human runs each case on a real
Windows build.

> **Execution status as of v1.1.0 release prep (2026-07-05): NOT RUN.** Every
> case below is still ☐ / NOT RUN — the manual GUI clickthrough was **not**
> performed in the automated release-prep session. The v1.1.0 backend paths are
> covered by automated unit + build-tagged integration tests
> (`internal/localfs` List/Home/Roots/Parent/Exists; `sftpx.DownloadPaths` +
> `Exists` in the integration suite), but the two-pane **GUI** behavior — pane
> rendering, drag-drop regression, overwrite dialogs, progress, and the
> `LocalParent` Wails multi-return — remains human-unverified. Do not treat
> v1.1.0 as GUI-QA-passed until these boxes are filled by a person. This tracks the SFTP two-pane feature (v1.1.0), built up across
commits: local browse API (commit 1), recursive remote→local download backend
(commit 2), **two-pane UI foundation (commit 3, this checklist's focus)**, and
transfer-action wiring (commit 4, later).

## Scope of commit 3 (what this round verifies)

Commit 3 adds the **two-pane UI foundation**: a read-only **local** pane
(`SftpPane.vue`) beside the existing **remote** pane (kept inline in
`SftpPanel.vue`). It does **not** wire cross-pane upload/download actions,
conflict handling, or any local destructive action (mkdir/rename/delete) — those
are out of scope here. The remote pane's existing upload/download/bookmarks/
preview/mkdir/rename/delete are unchanged and must not regress.

## Prerequisites

- A Windows build: `build-windows.bat` → `build\bin\ssh-terminal.exe`.
- A reachable SSH/SFTP server you control (throwaway host) for the remote pane.
- No real credentials, private keys, or real server addresses in any repo file
  or screenshot.

## Test cases

Legend: ☐ not run · ✅ pass · ❌ fail.

Run/version: `__________`  Date: `__________`  Tester: `__________`

### Panel + layout

| # | Case | Expected | Result |
|---|------|----------|--------|
| L1 | Open the SFTP panel (toolbar toggle) on a connected tab | Panel opens showing **two** panes: 本地 (local) and 远程 (remote). | ☐ |
| L2 | Wide window | Panes sit **side-by-side** (local left, remote right); the terminal remains usable. | ☐ |
| L3 | Narrow the window / SFTP region | Panes **stack** (local top, remote bottom) without overlap or clipping. | ☐ |
| L4 | Light and dark themes | Both panes, dividers, and the 本地/远程 tags render correctly in each theme. | ☐ |

### Local pane (read-only)

| # | Case | Expected | Result |
|---|------|----------|--------|
| LP1 | Local pane on open | Loads the **home** directory and lists its files/folders (folders first, sorted). | ☐ |
| LP2 | Double-click a local folder | Enters it; the path display updates. | ☐ |
| LP3 | Up (上一级) from a nested folder | Goes to the parent directory. | ☐ |
| LP4 | Up repeatedly to the drive root, then Up again | At the drive root, Up shows the **roots list** ("此电脑" with C:\ etc.); Up is then disabled/no-op (no loop, no error). | ☐ |
| LP5 | From the roots list, click into a drive (e.g. C:\) | Lists that drive's contents. | ☐ |
| LP6 | Home button | Returns to the home directory from anywhere. | ☐ |
| LP7 | Refresh | Reloads the current local directory (or roots). | ☐ |
| LP8 | A folder with no read permission / a deleted path | Shows a readable error, no crash; other navigation still works. | ☐ |
| LP9 | Empty folder | Shows "空目录" (empty), not an error. | ☐ |
| LP10 | No local destructive actions exist | Local pane offers no delete/rename/mkdir/upload controls (read-only by design in commit 3). | ☐ |

### Remote pane (must NOT regress)

| # | Case | Expected | Result |
|---|------|----------|--------|
| RP1 | Remote pane lists the remote cwd | Same behavior as v1.0.0. | ☐ |
| RP2 | Remote up / double-click directory / refresh | Work as before. | ☐ |
| RP3 | Remote bookmarks (add / jump / delete) | Work as before (saved-host tabs only; Quick Connect shows the hint). | ☐ |
| RP4 | Remote text preview (double-click a text file / right-click 预览) | Works, read-only. | ☐ |
| RP5 | Remote mkdir / rename / delete (with confirm) | Work as before; delete still confirms. | ☐ |
| RP6 | Remote upload button + per-file download button | Work as before, with the xfer progress bar. | ☐ |

### Lifecycle / regression (critical)

| # | Case | Expected | Result |
|---|------|----------|--------|
| X1 | Switch the active tab within a pane, then run a remote transfer | The `sftp:xfer:*` progress still shows for the **current** tab only; no progress bleeds across tabs (xfer bind/unbind lifecycle intact). | ☐ |
| X2 | **App.vue drag-drop**: drag files from the OS into the window | Still uploads to the remote cwd with the drop overlay + progress (the `app:filedrop` / `sftp:progress` path is unchanged). | ☐ |
| X3 | Open/close the SFTP panel repeatedly; open on multiple panes | No leaked listeners, no console errors, local pane re-loads home each open. | ☐ |

### Two-pane transfers (commit 4)

Select a row with a single click; the selected row is highlighted. **Upload →**
sends the selected **local** entry into the current **remote** directory;
**← Download** sends the selected **remote** entry into the current **local**
directory. Both buttons are disabled until a valid selection + destination
exist. Overwrite is confirmed via a dialog.

| # | Case | Expected | Result |
|---|------|----------|--------|
| T1 | Select a local file, note the remote cwd | Upload → becomes enabled; ← Download disabled until a remote row is selected. | ☐ |
| T2 | Upload → a local **file** | File appears in the remote pane after transfer; progress bar shown; remote pane refreshes. | ☐ |
| T3 | Upload → a local **folder** | Folder tree is uploaded recursively; remote pane refreshes. | ☐ |
| T4 | Upload → when the remote name already exists → **Cancel** | Overwrite dialog appears; Cancel does nothing (no transfer). | ☐ |
| T5 | Upload → when the remote name already exists → **Confirm (覆盖)** | Transfer proceeds and overwrites. | ☐ |
| T6 | Select a remote file, note the local cwd | ← Download becomes enabled. | ☐ |
| T7 | ← Download a remote **file** | File appears in the local pane after transfer; local pane refreshes. | ☐ |
| T8 | ← Download a remote **folder** | Folder tree is downloaded recursively into local cwd; local pane refreshes. | ☐ |
| T9 | ← Download when the local name already exists → **Cancel** | Overwrite dialog appears; Cancel does nothing. | ☐ |
| T10 | ← Download when the local name already exists → **Confirm (覆盖)** | Transfer proceeds and overwrites. | ☐ |
| T11 | Progress bar | Shows during both upload and download, then clears on completion. | ☐ |
| T12 | Transfer failure (e.g. permission denied) | A readable error is shown; no crash. | ☐ |
| T13 | Buttons disabled at roots | With the local pane at the roots list ("此电脑"), Upload →/← Download are disabled (no valid local cwd/selection). | ☐ |
| T14 | Old remote actions still work | Remote header upload, per-row download, mkdir, rename, delete-with-confirm, bookmarks, preview all still function. | ☐ |
| T15 | **App.vue drag-drop still works** | Dragging OS files into the window still uploads to the remote cwd via the unchanged `app:filedrop` path. | ☐ |
| T16 | Tab switch during/after a transfer | xfer progress stays scoped to the correct tab; no cross-tab bleed. | ☐ |
| T17 | **LocalParent multi-return on a real Windows build** | Local "up" navigation works correctly at every level incl. drive root (verifies the Wails `(string, bool)` marshalling in practice). | ☐ |

## Notes for the tester

- Do not mark the suite "passed" unless every case was actually executed by a
  human on a real build. Skipped cases stay ☐.
- No secrets, real credentials, real private keys, or real server addresses in
  any repo file or screenshot.
