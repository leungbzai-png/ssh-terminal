# VPS Monitor Sidebar — Manual QA Checklist

**Status of this document:** *authored, not yet human-executed.* No case below is
marked PASS. Fill in Pass/Fail only after a human runs each case on a real
Windows build against a real Linux host.

> **Execution status as of v1.2.0 release prep (2026-07-05): NOT RUN.** Every
> case below is still ☐ / NOT RUN — the manual GUI clickthrough was **not**
> performed in the automated release-prep session. The v1.2.0 backend paths are
> covered by automated tests (`internal/sysmon` parsers + CPU-delta unit tests;
> a build-tagged `internal/sshsess` integration test that drives `Manager.Run`
> end to end, incl. the `Run → sysmon.ParseAll` pipeline). But the live monitor
> **GUI** behavior — panel rendering, sparklines, interval switching, the
> disconnected / unsupported / loading states, per-tab isolation, and timer
> cleanup — remains human-unverified. Do not treat v1.2.0 as GUI-QA-passed until
> these boxes are filled by a person.

This tracks the VPS Monitor Sidebar (v1.2.0), built across commits: `sysmon`
parser package (commit 1), `MonitorSample` API over SSH exec (commit 2), the
sidebar + sparkline UI shell (commit 3), and live per-tab polling (commit 4).

## What v1.2.0 is (and is NOT)

Agentless, **Linux-only**, read-only monitoring over the existing SSH session:
CPU %, memory %, swap % (when present), disk usage for `/`, load average
(1/5/15m), uptime, and CPU/memory sparklines. One compact fixed command per
sample on a **separate SSH channel** (never the interactive shell). **Not** in
scope: remote agent/daemon, sudo, remote install, multi-server dashboard,
historical DB, alerting, process/top clone, per-NIC graph, non-Linux remote
hosts, or any metrics persistence.

## Prerequisites

- A Windows build: `build-windows.bat` → `build\bin\ssh-terminal.exe`.
- A reachable **Linux** SSH host you control (throwaway) for the live cases.
- If available, a **non-Linux** SSH host (e.g. a BSD/macOS/router shell) to
  verify the unsupported state — otherwise mark U1 as N/A.
- No real credentials, private keys, or real server addresses in any repo file
  or screenshot.

## Test cases

Legend: ☐ not run · ✅ pass · ❌ fail.

Run/version: `__________`  Date: `__________`  Tester: `__________`

### Panel + layout

| # | Case | Expected | Result |
|---|------|----------|--------|
| L1 | Click the monitor toggle (toolbar activity icon) on a connected tab | The **left** monitor sidebar opens (VPS 监控), left of the terminal, right of the host sidebar. | ☐ |
| L2 | Toggle it off | Sidebar closes; the terminal reflows to fill the space (xterm refits, no clipped rows/cols). | ☐ |
| L3 | Open the monitor **and** the SFTP panel together | Layout is monitor (left) · terminal (center) · SFTP (right); all three usable, no overlap. | ☐ |
| L4 | Light and dark themes | Cards, bars, sparklines, and text render correctly in both themes (toggle theme while open). | ☐ |
| L5 | Narrow the window | Monitor column keeps a sensible min width; terminal stays usable; no horizontal page scroll. | ☐ |

### Live metrics (connected Linux host)

| # | Case | Expected | Result |
|---|------|----------|--------|
| M1 | Open monitor on a connected Linux tab | Within one interval the cards populate: CPU, 内存 (memory), Swap, 磁盘 / (disk), plus 负载 (load) and 运行 (uptime). | ☐ |
| M2 | CPU first reading | The **first** CPU value shows "测量中…" (measuring), then resolves to a percentage on the next sample. | ☐ |
| M3 | CPU under load | Run `yes > /dev/null &` (then `kill`) on the host; CPU % rises then falls; the CPU sparkline reflects the change. | ☐ |
| M4 | Memory | 内存 % and its bar/sparkline look plausible vs `free -m` on the host. | ☐ |
| M5 | Swap present | On a host **with** swap, Swap shows a % and bar. | ☐ |
| M6 | Swap absent | On a host with **no** swap, Swap shows "无" (none), no bar, no crash. | ☐ |
| M7 | Disk / | 磁盘 / % and "used / size" match `df -P /` on the host (within rounding). | ☐ |
| M8 | Load + uptime | 负载 shows 1/5/15m values matching `uptime`; 运行 shows a sensible duration. | ☐ |
| M9 | Sparklines grow | CPU and memory sparklines accumulate points over time and cap (old points scroll off), staying smooth. | ☐ |

### Interval selector

| # | Case | Expected | Result |
|---|------|----------|--------|
| I1 | Default interval | A fresh monitor defaults to **5s**; the 5s button is highlighted. | ☐ |
| I2 | Switch to 2s | Sampling visibly speeds up (watch the clock/updates); no stacked/overlapping requests, no UI stutter. | ☐ |
| I3 | Switch to 10s | Sampling slows to ~10s cadence. | ☐ |
| I4 | Interval is per-tab | Set tab A to 2s and tab B to 10s; each tab keeps its own interval when you switch between them. | ☐ |

### States (disconnected / unsupported / error)

| # | Case | Expected | Result |
|---|------|----------|--------|
| S1 | Open monitor on an **idle** (restored, not connected) tab | Shows "无活动会话" (no active session), no spinner loop, no polling. | ☐ |
| S2 | Disconnect the session (type `exit`, or drop the network) while the monitor is open | Panel switches to the no-session state; metrics stop updating; no error spam. | ☐ |
| S3 | Reconnect the tab | Monitoring resumes fresh (CPU shows "测量中…" again on the first new sample; sparklines restart). | ☐ |
| U1 | Open monitor on a **non-Linux** host (if available) | Shows the "不支持的主机" (unsupported) state naming the detected OS; no metric cards, no crash. (N/A if no such host.) | ☐ |

### Per-tab isolation + lifecycle (critical)

| # | Case | Expected | Result |
|---|------|----------|--------|
| P1 | Two connected tabs, monitor open on both, then switch between them | Each tab shows **its own** metrics and sparkline history; no data bleeds across tabs. | ☐ |
| P2 | Switch away from a monitored tab and back | Its previous reading + sparkline history are still there (state persisted per tab), then refreshes. | ☐ |
| P3 | Close a monitored tab | No errors; polling for that tab stops (no background monitoring); its state is dropped. | ☐ |
| P4 | Open/close the monitor panel repeatedly | No leaked timers or console errors; each reopen resumes cleanly. | ☐ |
| P5 | Leave the monitor running for several minutes | No memory growth from unbounded history (buffer is capped), no slowdown, terminal stays responsive. | ☐ |

### Non-interference + security

| # | Case | Expected | Result |
|---|------|----------|--------|
| N1 | Type in the terminal while the monitor polls (2s) | Keystrokes/output are unaffected; monitoring runs on a separate channel and does not inject into the shell. | ☐ |
| N2 | Run an SFTP transfer with the monitor open | Transfer and its progress are unaffected; monitor keeps updating. | ☐ |
| N3 | No persistence | After using the monitor, confirm **no** monitor data is written under `data/` (no new file; `settings.json`/`session.json`/`bookmarks.json` unchanged by monitoring). | ☐ |
| N4 | No secrets on the wire | (Spot check) The monitor command is a fixed `/proc`/`df`/`uname` string with no credentials interpolated. | ☐ |

## Notes for the tester

- Do not mark the suite "passed" unless every case was actually executed by a
  human on a real build against a real host. Skipped cases stay ☐.
- No secrets, real credentials, real private keys, or real server addresses in
  any repo file or screenshot.
- Distro variance is expected; if a field parses oddly on an unusual distro,
  record the distro and the raw `/proc`/`df` line (with no host identifiers).
