# Workspace Resizable Splitters — Manual QA Checklist

**Status of this document:** *authored, not yet human-executed.* No case below is
marked PASS. Fill in Pass/Fail only after a human runs each case on a real
Windows build.

> **Execution status as of the v1.2.1 UI-polish patch: NOT RUN.** Every case
> below is still ☐ / NOT RUN — this is a GUI-only change (drag splitters, xterm
> reflow, layout persistence) that cannot be exercised by `npm run build` or the
> Go tests. The automated gate (build + typecheck + Go suites) is green, but the
> resize **GUI** behavior is human-unverified. Do not treat this patch as
> GUI-QA-passed until these boxes are filled by a person.

## What this patch changes

Adds draggable splitters between the **VPS monitor** (left), the **terminal**
(center), and the **SFTP** panel (right), replacing the previous fixed/clamped
column widths. Widths are user-adjustable, clamped to safe min/max, scaled down
on narrow windows so the terminal stays usable, and persisted locally
(`localStorage`: `ssh-terminal.monitorWidth`, `ssh-terminal.sftpWidth` — integer
pixels only). Double-click a splitter to reset that panel to its default width.
No backend, monitor, or SFTP feature change.

## Prerequisites

- A Windows build: `build-windows.bat` → `build\bin\ssh-terminal.exe`.
- A reachable Linux SSH host (for the monitor + SFTP cases).
- No real credentials, private keys, or real server addresses in any repo file
  or screenshot.

## Test cases

Legend: ☐ not run · ✅ pass · ❌ fail.

Run/version: `__________`  Date: `__________`  Tester: `__________`

### Layout combinations

| # | Case | Expected | Result |
|---|------|----------|--------|
| C1 | Monitor closed, SFTP closed | Terminal fills the pane; no splitters visible. | ☐ |
| C2 | Monitor open, SFTP closed | One splitter between monitor and terminal; dragging it adjusts the monitor width; the terminal takes the rest. | ☐ |
| C3 | Monitor closed, SFTP open | One splitter between terminal and SFTP; dragging it adjusts the SFTP width. | ☐ |
| C4 | Monitor open, SFTP open | Layout is Monitor \| Terminal \| SFTP with **two** splitters; both drag independently. | ☐ |

### Drag behavior + constraints

| # | Case | Expected | Result |
|---|------|----------|--------|
| D1 | Drag the monitor splitter right/left | Monitor width grows/shrinks, clamped ~180–360px; cannot collapse the panel or the terminal below usable width. | ☐ |
| D2 | Drag the SFTP splitter left/right | SFTP width grows/shrinks (drag left = wider), clamped ~360px min; terminal keeps ≥ ~360px. | ☐ |
| D3 | Drag quickly / far past the edge | No overshoot jump, no runaway; width stops at the clamp; no horizontal page scrollbar appears. | ☐ |
| D4 | Grab a splitter on a **narrow** window (panels already scaled) | The handle does not jump on first move; dragging feels anchored to the current position. | ☐ |
| D5 | Cursor + visual | Hovering a splitter shows the `col-resize` cursor and a subtle accent line; the line strengthens while dragging; no thick/ugly bar; does not cover terminal input. | ☐ |
| D6 | Double-click a splitter | That panel resets to its default width. | ☐ |
| D7 | Light and dark themes | Splitter hover/active line looks correct in both themes. | ☐ |

### xterm reflow

| # | Case | Expected | Result |
|---|------|----------|--------|
| X1 | Terminal reflows after dragging either splitter | The terminal re-fits to its new width (columns/rows update); it does **not** keep old dimensions or clip. | ☐ |
| X2 | Type/run a full-width command (e.g. `htop`, `ls` wide) after a drag | Output wraps to the new width correctly. | ☐ |
| X3 | Rapid drag | No visible stutter; reflow keeps up (rAF-throttled), terminal stays responsive. | ☐ |

### Persistence

| # | Case | Expected | Result |
|---|------|----------|--------|
| P1 | Set custom monitor/SFTP widths, close and reopen the app | Widths restore to the last values (from localStorage). | ☐ |
| P2 | Double-click reset, reopen | Reset widths persist across restart. | ☐ |
| P3 | Inspect persisted data | Only integer pixel widths under `ssh-terminal.monitorWidth` / `ssh-terminal.sftpWidth`; **no** local/remote paths, hostnames, IPs, usernames, credentials, monitor samples, or SFTP listings anywhere. | ☐ |

### Regression (must NOT break)

| # | Case | Expected | Result |
|---|------|----------|--------|
| R1 | VPS monitor polling | Metrics still refresh on interval after dragging the monitor splitter. | ☐ |
| R2 | Monitor enable/disable + interval buttons (2/5/10s) | Still work; closing the monitor removes its splitter and the terminal reclaims the space. | ☐ |
| R3 | Terminal input/output | Unaffected by splitters; keystrokes/output normal. | ☐ |
| R4 | SFTP local + remote panes | Both render and function after a drag; side-by-side/stacked switch still works at narrow SFTP widths. | ☐ |
| R5 | SFTP upload/download | Still work after resizing. | ☐ |
| R6 | App.vue drag-drop upload | Dragging OS files into the window still uploads to the remote cwd. | ☐ |
| R7 | Tab switching | Switching tabs keeps the layout intact; per-tab monitor/SFTP visibility still drives which splitters show. | ☐ |
| R8 | Split panes (向右分屏) | Multiple panes each render splitters; shared widths apply consistently; no crash. | ☐ |
| R9 | Window resize | Resizing the OS window does not destroy the layout or cause horizontal overflow; panels scale down to protect the terminal minimum. | ☐ |

## Notes for the tester

- Do not mark the suite "passed" unless every case was actually executed by a
  human on a real build. Skipped cases stay ☐.
- No secrets, real credentials, real private keys, or real server addresses in
  any repo file or screenshot.
