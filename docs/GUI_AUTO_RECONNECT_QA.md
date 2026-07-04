# GUI Auto-Reconnect — Manual QA Checklist

**Status of this document:** *authored, not yet human-executed.* No case below is
marked PASS. It exists to make the one remaining pre-expansion QA gap explicit
and repeatable. Fill in Pass/Fail only after a human actually runs each case.

## Why this is a manual checklist (not an automated test)

The auto-reconnect **backend close signal** — i.e. whether an unexpected drop
versus a user-initiated close produces the correct close `reason`, which is what
the frontend keys on — **is already covered by the build-tagged integration
tests** (`internal/sshsess/integration_test.go`,
`TestIntegrationAdvancedSSH/ReconnectCloseSignal/{UnexpectedDrop,UserClose}`).
See `docs/INTEGRATION_TESTS.md`.

What is **not** automated is the **GUI / WebView2 behavior**: the reconnect
"burst" timing, the on-screen status text, the attempt counter, the cap message,
the cancel button, and the overlay state transitions. That logic lives as
component-local state inside `frontend/src/components/Terminal.vue`
(`scheduleReconnect` / `attemptReconnect` / `onUnexpectedClose` /
`cancelAutoReconnect`). Exercising it faithfully needs a running WebView2 window
and a real SSH session that can be dropped and restored. There is currently no
frontend test runner, and extracting this state into a pure, unit-testable module
would be a component refactor that is deliberately out of scope for the v1.0.0
stabilization / pre-v1.1.0 readiness pass. **This checklist therefore covers
GUI/WebView2 behavior only.**

## Prerequisites

- A Windows build of the app: `build-windows.bat` → `build\bin\ssh-terminal.exe`.
- A reachable SSH server you control and may safely disconnect at will.
- The ability to interrupt connectivity to that server on demand (see setup).

## Test host setup recommendations

Use a **disposable, throwaway** SSH endpoint — never a production host:

- A local VM or a local container exposing SSH on `127.0.0.1:<port>`, **or**
- A personal test VPS you are comfortable killing the sshd on.

Ways to force an *unexpected* drop (not a clean logout):

- Stop/kill sshd on the server (`systemctl stop ssh`), or
- Block the port with a firewall rule mid-session, or
- Suspend/pause the VM, or drop the VM's network adapter.

Ways to produce a *clean* close (must NOT reconnect):

- Type `exit` in the remote shell (clean exit), or
- Close the tab from the app (user close).

**Auto-reconnect config** is per host, under the host's **高级 SSH → 自动重连**
panel (`AutoReconnect`: `enabled`, `maxAttempts` 0–10, `delaySeconds` 1–60).
For fast iteration, set `maxAttempts = 2` and `delaySeconds = 2`.

## Safety notes

- **Do not** put real credentials, private keys, passphrases, or real server
  addresses into any file in this repository, into screenshots committed to the
  repo, or into this checklist. Record only Pass/Fail and neutral notes.
- Prefer SSH **key** auth against a throwaway host. If you must use a password,
  use a throwaway one and do not write it down here.
- `data/` (hosts.json, secret.key, known_hosts, session.json, bookmarks.json) is
  git-ignored and must never be committed. Do not paste its contents here.

## Test cases

Legend: ☐ not run · ✅ pass · ❌ fail. Record the app version and date at the top
of each run.

Run/version: `__________`  Date: `__________`  Tester: `__________`

| # | Case | Steps | Expected result | Result |
|---|------|-------|-----------------|--------|
| G1 | Unexpected drop triggers a reconnect burst | Connect with auto-reconnect **enabled** (`maxAttempts=2`, `delaySeconds=2`). Force an unexpected drop (kill sshd / block port). | Terminal shows "session closed: …", then a status line "将在 2s 后自动重连 (1/2)…", then "自动重连中… (1/2)". A reconnect attempt is actually made. | ☐ |
| G2 | Successful reconnect resets state | During G1, restore the server before the cap is hit. | A subsequent attempt reconnects; the shell is live again; the auto-status line clears; the attempt counter resets (a later drop starts again at 1/N, not 2/N). | ☐ |
| G3 | maxAttempts caps the burst | With the server kept **down**, let the burst run to the cap. | After `maxAttempts` failed attempts, a message "自动重连已达上限（N 次），已停止" appears and no further attempts are scheduled. The tab rests in a disconnected/error state, not a spinning loop. | ☐ |
| G4 | Cancel stops a pending burst | Trigger a drop so a "将在 Ns 后自动重连…" countdown is showing, then click **取消** (cancel). | The countdown stops immediately; no further attempt fires; the auto-status line clears. | ☐ |
| G5 | Manual reconnect supersedes the burst | During an active burst, click **重新连接** (manual reconnect) after restoring the server. | Any pending auto attempt is cancelled (no double connect); a single manual reconnect runs; on success the shell is live and the counter is reset. | ☐ |
| G6 | User close does NOT reconnect | With auto-reconnect enabled and a live session, close the tab. | No reconnect is attempted (close reason is "user closed"). | ☐ |
| G7 | Clean exit does NOT reconnect | With auto-reconnect enabled and a live session, type `exit` on the remote. | No reconnect is attempted (clean exit → empty reason). | ☐ |
| G8 | Auth failure does not infinite-loop | Enable auto-reconnect; after connecting, change/revoke the credential server-side so re-auth fails, then force a drop. | Reconnect attempts fail and stop at `maxAttempts` (does **not** loop forever). NOTE: by design the frontend cannot distinguish auth-failure from a network drop, so it will retry up to the cap and then stop — this is expected, not a bug. | ☐ |
| G9 | Disabled config never reconnects | Set auto-reconnect **disabled** on the host; force an unexpected drop. | No reconnect attempt; the tab simply shows disconnected with a manual "重新连接" button. | ☐ |
| G10 | Unmount cancels a pending burst | Trigger a drop so a countdown is showing, then close the tab (or the app). | The pending timer is cleared on unmount; no orphaned reconnect fires afterward; no console error. | ☐ |

## Pass/Fail recording format

For each run, copy the table, fill the Result column with ✅/❌, and add a short
note for any ❌ (observed vs expected). Do **not** mark the suite "passed" unless
every case was actually executed by a human on a real build. If a case was
skipped, leave it ☐ and say so — never infer PASS from the backend integration
tests, which cover the close *signal* only, not the GUI.

## Known limitation carried forward

Until this checklist has been executed by a human, the project's honest status is:
**GUI auto-reconnect behavior is reviewed in code and its backend close signal is
covered by integration tests, but the end-to-end GUI/WebView2 reconnect UX has
not been separately human-verified.**
