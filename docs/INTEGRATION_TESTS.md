# Integration Tests — Advanced SSH (backend-live)

The `internal/sshsess` package carries a **build-tagged integration suite** that
drives the real `sshsess.Manager` against **disposable, in-process SSH servers**
on `127.0.0.1`. It is the committed, regression-safe successor to the temporary
`qa-local/sshqa/` harness that was used to verify the v0.9.0 Advanced SSH code
paths.

These are **backend-live** tests — they exercise the actual dial / ProxyJump /
port-forward / SOCKS / diagnostics / close-signal code — but they are **not** a
GUI substitute. The Vue-side auto-reconnect cap/cancel logic still requires
manual QA (see *Known limitations*).

---

## What they cover

| Area | Subtest | What it asserts |
|------|---------|-----------------|
| A. ProxyJump / Bastion | `ProxyJumpBastion` | Connects a target **through** an in-process bastion using the real ProxyJump path; session opens, `ActiveCount()==1`, closes cleanly to `0`. |
| B. Local forwarding | `LocalForward` | Local forward on the default `127.0.0.1` bind carries HTTP traffic to a target-side service; listener is released after close. |
| B′. Occupied local port | `LocalForwardOccupiedPort` | A bind clash does **not** abort the session (`ActiveCount()==1`); a readable, `DiagForward`-classified error is reported via `TunnelStatus.Err`. |
| C. Remote forwarding | `RemoteForward` | Remote (reverse) forward carries traffic from the server side back to a local service; the remote listener is released after close. |
| D. Dynamic SOCKS5 | `DynamicSOCKS` | Dynamic forward on the default bind proxies a SOCKS5 CONNECT (domain address form) to a target service; listener is released after close. |
| E. Connection diagnostics | `Diagnostics` | `classifyError` maps live failures: TCP refused → `DiagTCP`, wrong password → `DiagAuth`, `.invalid` host → `DiagDNS`, garbage key file → `DiagKey`, closed bastion → `DiagProxyJump`. Each yields a non-empty readable message. |
| F. Runtime cleanup | `CleanupCloseAll` + B/C/D | Per-tunnel listener release is asserted in B/C/D via `waitPortFree`; `CloseAll` drains `ActiveCount()` to `0`. |
| G. Auto-reconnect close signal | `ReconnectCloseSignal/{UnexpectedDrop,UserClose}` | An unexpected drop fires a **non-user** close reason (reconnect-eligible); a user close fires the distinguishable `"user closed"` reason. |

### Safety properties enforced by the tests themselves

- All credentials (ed25519 host keys, passwords, sentinels) are **generated at
  runtime**; nothing static is committed.
- No password, private key, or PEM block is ever printed — error text is routed
  through `internal/redact`.
- Every server, HTTP service, and listener is torn down via `t.Cleanup`.
- Each subtest resets the per-process `known_hosts` file so connections are
  first-seen and deterministic (avoids an ephemeral-port-reuse host-key
  mismatch across the run).

---

## How to run

Integration tests are **excluded** from the normal test run. The normal command
is unchanged and stays fast and offline:

```bash
go test ./...
```

Run the integration suite with the `integration` build tag:

```bash
go test -tags=integration ./...
```

Package-specific (recommended while iterating):

```bash
go test -tags=integration ./internal/sshsess -run Integration -v
```

They need only localhost, no admin privileges, no Docker/WSL, and no real
server or secret.

---

## Known limitations

- **GUI auto-reconnect still needs manual QA.** These tests confirm only the
  backend close *signal* the frontend keys on. The Vue-side reconnect burst
  (capped attempts, unexpected-drop-only, cancellable) is not exercised here —
  verify it manually before v1.0.0 (see `docs/QA_v0.8.0_v0.9.0.md`).
- **Backend-live, not full end-to-end.** No WebView2 / xterm.js / event bridge
  is involved.
- **`go test -race` requires CGO/GCC on Windows.** The suite is race-clean by
  construction (shared tunnel state is mutex-guarded), but running with `-race`
  needs a gcc toolchain that is not assumed to be present here.

---

## Where the code lives

```
internal/sshsess/
├── integration_test.go          // TestIntegrationAdvancedSSH + subtests
└── integration_server_test.go   // disposable in-process SSH server + localhost helpers
```

Both files begin with:

```go
//go:build integration
// +build integration
```

so they compile only under `-tags=integration`.
