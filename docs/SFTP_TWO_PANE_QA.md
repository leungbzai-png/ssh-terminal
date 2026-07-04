# SFTP Two-Pane — Manual QA Checklist

**Status of this document:** *authored, not yet human-executed.* No case below is
marked PASS. Fill in Pass/Fail only after a human runs each case on a real
Windows build. This tracks the SFTP two-pane feature (v1.1.0), built up across
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

## Notes for the tester

- The local pane is **read-only** in this build; it browses only. Cross-pane
  Upload →/← Download wiring lands in commit 4 and has its own checklist rows.
- Do not mark the suite "passed" unless every case was actually executed by a
  human on a real build. Skipped cases stay ☐.
