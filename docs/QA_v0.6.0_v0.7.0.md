# Manual QA Checklist — v0.6.0 (Terminal UX) + v0.7.0 (SFTP UX)

Build under test: `qa-local/ssh-terminal-v0.6.0-v0.7.0-qa/ssh-terminal.exe`
Run from inside that folder so `data/` is created there. **Do not** copy any real
`data/`, `secret.key`, `hosts.json`, keys, or logs into the QA folder.

Legend: [ ] not run · [x] pass · [!] fail (note details)

---

## v0.6.0 — Terminal UX

### A. Terminal search
- [ ] Connect a session; press Ctrl+F — search box opens
- [ ] Type a term present in the buffer — match indicator shows index/total
- [ ] Enter = next match, Shift+Enter = previous match
- [ ] Type a term NOT present — shows 无匹配, no crash
- [ ] Esc closes the search box; typing goes to the terminal again
- [ ] With two tabs, search only affects the current tab
- [ ] Disconnect the session, then open/close search — no crash/panic

### B. Font family
- [ ] Settings → 字体: pick a preset from the datalist — terminal applies it
- [ ] Enter a nonsense font name — terminal falls back, no crash
- [ ] Restart app — font family persists

### C. Font size
- [ ] Settings → 字号: change within 8–32 — terminal updates
- [ ] Ctrl+= / Ctrl+- in the terminal increase/decrease size; Ctrl+0 resets to 14
- [ ] Enter an out-of-range value (e.g. 999) and save — clamped to 32, no crash
- [ ] Restart app — font size persists
- [ ] With multiple tabs, size change applies consistently

### D. Tab restore
- [ ] Open 2–3 saved-host tabs, connect at least one, close the app
- [ ] Relaunch — the saved-host tabs reappear as **idle** (Ready to connect), NOT auto-connected
- [ ] Click 连接 on a restored tab — it connects normally
- [ ] Open a Quick Connect tab, close the app, relaunch — the Quick Connect tab is NOT restored
- [ ] Confirm `data/session.json` contains only hostId/hostName (no secrets)
- [ ] Delete a host that had a restored tab, relaunch — that tab is skipped, no crash
- [ ] Confirm restore works with a v0.5.0-style hosts.json (no migration error)

### E. Keyboard shortcut help
- [ ] Open via the sidebar help (?) button — dialog lists shortcuts + mouse actions
- [ ] Press F1 — dialog toggles open/closed
- [ ] Listed shortcuts match reality (search, font size, F1) — nothing invented
- [ ] Closing the dialog returns focus; terminal input still works

---

## v0.7.0 — SFTP UX

### F. Upload progress
- [ ] Open SFTP; click upload, pick a sizeable file — footer shows 上传 % + filename
- [ ] On success the listing refreshes
- [ ] Trigger a failure (e.g. no permission) — error shown, no crash

### G. Download progress
- [ ] Right-click a file → 下载, choose a local path — footer shows 下载 % + filename
- [ ] File is written to the chosen local path
- [ ] Cancel the save dialog — nothing happens, no crash
- [ ] **Multi-tab isolation:** open two SSH tabs in the SAME pane, both with SFTP open; start a download in tab A, switch to tab B mid-transfer — progress tracks the correct tab and does NOT bleed into the other tab's panel

### H. Drag-upload polish
- [ ] Drag a file over the window while connected — overlay shows accept + target dir
- [ ] Drag with no connected session — overlay shows a reject state
- [ ] Drop a valid file — it uploads to the shown directory
- [ ] Drag a non-file item — overlay does not falsely accept; no crash

### I. Remote bookmarks
- [ ] In SFTP, open the bookmark (★) menu → 添加当前路径, name it — it appears
- [ ] Navigate elsewhere, click the bookmark — jumps back to that path
- [ ] Delete a bookmark — it is removed and stays removed after restart
- [ ] Restart app — bookmarks persist (per host)
- [ ] On a Quick Connect tab, the bookmark menu shows a not-supported hint (no crash)
- [ ] Confirm `data/bookmarks.json` has only name/path/hostId (no secrets)

### J. Text preview
- [ ] Double-click a small `.txt`/`.log`/`.md` file — read-only preview opens
- [ ] Right-click any file → 预览 — preview opens
- [ ] Preview a large file (> 512 KB) — shows "文件过大", does not freeze the UI
- [ ] Preview a binary file (e.g. the exe, an image) — shows "不是文本", no garbage flood
- [ ] Preview does not modify the remote file (re-list; size/mtime unchanged)
- [ ] Preview of a missing/permission-denied file shows an error, no crash

---

## K. Security regression
- [ ] `hosts.json` still stores only encrypted secrets (`encPassword`/`encPassphrase`)
- [ ] No plaintext password/passphrase/private key anywhere under `data/`
- [ ] Quick Connect secrets are not written to disk (checked in session.json + hosts.json)
- [ ] Safe host export still excludes password/passphrase/private key/.key.enc/secret.key
- [ ] `data/session.json` and `data/bookmarks.json` contain only non-secret fields
- [ ] Imported private key is still only stored as `.key.enc`
- [ ] `data/secret.key` location/format unchanged
- [ ] Backward compat: a v0.5.0 `data/` folder loads without error

Suggested scan (PowerShell, from the QA folder):
```powershell
Get-ChildItem -Recurse .\data\ -File | ForEach-Object {
  Select-String -Path $_.FullName -Pattern 'PRIVATE KEY','BEGIN OPENSSH','"password"','"passphrase"'
}
```

## L. Build / artifact
- [ ] `go test ./...` passes (developer machine)
- [ ] `build-windows.bat` passes
- [ ] QA build launches from `qa-local/…`
- [ ] No release published this round (no tag, no push, no GitHub Release)

---

## Notes / findings
(record failures, screenshots, follow-ups here)
