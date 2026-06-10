# Release Process — SSH Terminal

Step-by-step guide for creating a new release. Follow this every time.

---

## Pre-Release Checklist

Before starting the release:

- [ ] All planned features for this version are merged to `main`
- [ ] No open Critical or High issues for this version
- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go mod verify` passes
- [ ] Manual QA completed per `docs/QA_CHECKLIST.md` (minimum: S-01, S-10, F-01, D-01, C-01)
- [ ] `CHANGELOG.md` updated with new version section
- [ ] Version string in `app.go:AppInfo()` updated
- [ ] Version in `wails.json` (productVersion) updated
- [ ] Version in `frontend/package.json` updated
- [ ] All three version numbers match

---

## Step 1: Update Version Numbers

Edit these three files — all must show the same version (e.g., `0.3.0`):

**`app.go`** (~line 116):
```go
"version": "0.3.0",
```

**`wails.json`**:
```json
"productVersion": "0.3.0"
```

**`frontend/package.json`**:
```json
"version": "0.3.0"
```

---

## Step 2: Update CHANGELOG.md

Add a new section at the top of `CHANGELOG.md`:

```markdown
## [0.3.0] - YYYY-MM-DD

### Added
- ...

### Fixed
- ...

### Changed
- ...
```

---

## Step 3: Commit Release Preparation

```powershell
cd "E:\Projects\Active\ssh-terminal"
git add app.go wails.json frontend/package.json CHANGELOG.md
git commit -m "release: v0.3.0"
git push
```

---

## Step 4: Build the Release Exe

```powershell
cd "E:\Projects\Active\ssh-terminal"
.\build-windows.bat
```

Expected output:
```
=== Done ===
Output: build\bin\ssh-terminal.exe
```

After the build, verify git is still clean:
```powershell
git status
```

If `frontend/package-lock.json` shows as modified (npm install side effect), commit it:
```powershell
git add frontend/package-lock.json
git commit -m "chore: update package-lock after build"
git push
```

---

## Step 5: Create the Release Zip

```powershell
$version = "0.3.0"
$zipPath = "E:\Backup\Releases\ssh-terminal-v$version-windows-amd64.zip"
$tmpDir  = "E:\Backup\Releases\ssh-terminal-v$version-windows-amd64"

New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
Copy-Item "build\bin\ssh-terminal.exe" "$tmpDir\ssh-terminal.exe"
Copy-Item "README.md" "$tmpDir\README.md"
Copy-Item "LICENSE" "$tmpDir\LICENSE"

Compress-Archive -Path "$tmpDir\*" -DestinationPath $zipPath -Force
Remove-Item $tmpDir -Recurse -Force

$zip = Get-Item $zipPath
Write-Output "Created: $($zip.FullName) ($([math]::Round($zip.Length/1MB,2)) MB)"
```

Verify zip contents:
```powershell
Add-Type -AssemblyName System.IO.Compression.FileSystem
$z = [System.IO.Compression.ZipFile]::OpenRead($zipPath)
$z.Entries | Select-Object FullName, @{N='KB';E={[math]::Round($_.Length/1KB,1)}}
$z.Dispose()
```

Expected:
```
FullName          KB
--------          --
LICENSE            1
README.md         ...
ssh-terminal.exe  ...
```

---

## Step 6: Create Git Tag

```powershell
git tag -a "v0.3.0" -m "SSH Terminal v0.3.0"
git push origin "v0.3.0"
```

---

## Step 7: Create GitHub Release

1. Go to: `https://github.com/leungbzai-png/ssh-terminal/releases/new`
2. **Tag:** Select `v0.3.0`
3. **Title:** `SSH Terminal v0.3.0`
4. **Description:** Copy the relevant section from `CHANGELOG.md` and the release notes template from `docs/GITHUB_RELEASE.md`
5. **Attach file:** Upload `E:\Backup\Releases\ssh-terminal-v0.3.0-windows-amd64.zip`
6. Check: **Set as the latest release**
7. Click: **Publish release**

---

## Step 8: Post-Release

- [ ] Update `docs/SESSION_STATUS.md`:
  - Set **Current Version** to the new version
  - Move the old version to **Release Status** table
  - Update **Completed Work** section
  - Update **Known Issues** (close fixed ones)
  - Update **Next Development Direction**
- [ ] Commit and push the status update:
  ```powershell
  git add docs/SESSION_STATUS.md
  git commit -m "docs: update session status for v0.3.0"
  git push
  ```

---

## Rollback / Hotfix Procedure

If a Critical bug is found after release:

1. Create a hotfix branch: `git checkout -b hotfix/v0.3.1`
2. Fix the bug, commit: `git commit -m "fix: <description>"`
3. Bump patch version (e.g., `0.3.0` → `0.3.1`) in all three version files
4. Add entry to `CHANGELOG.md`
5. Push branch, merge to `main` via PR (or direct push for solo project)
6. Follow Steps 4–8 above with the new version number

---

## Version Numbering Convention

`MAJOR.MINOR.PATCH`

| Change | Bump |
|--------|------|
| Breaking change to `hosts.json` / `settings.json` format | MAJOR |
| New feature, backward-compatible | MINOR |
| Bug fix only | PATCH |
| Documentation / tooling only | No bump needed (no release) |

Until v1.0.0, treat MINOR as the primary indicator of "what's new."
