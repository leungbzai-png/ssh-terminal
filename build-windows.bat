@echo off
setlocal

REM ============================================================
REM  SSH Terminal - Windows build script
REM  Run from the project root directory.
REM  Requires: Go 1.22+, Node.js 18+, Wails CLI v2.12+
REM ============================================================

echo.
echo === Checking Go ===
where go >nul 2>nul
if errorlevel 1 (
  echo [!] Go not found in PATH. Install Go 1.22+ and ensure it is on PATH.
  exit /b 1
)
go version

echo.
echo === Checking Node.js ===
where node >nul 2>nul
if errorlevel 1 (
  echo [!] Node.js not found. Install Node 18+ first: https://nodejs.org
  exit /b 1
)
node --version

echo.
echo === Checking / installing Wails CLI ===
where wails >nul 2>nul
if errorlevel 1 (
  echo Installing wails CLI...
  go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
  if errorlevel 1 (
    echo [!] Wails install failed
    exit /b 1
  )
  set "PATH=%USERPROFILE%\go\bin;%PATH%"
)
wails version

echo.
echo === Building ===
wails build -clean -trimpath -ldflags "-s -w" -o ssh-terminal.exe
if errorlevel 1 (
  echo [!] Build failed
  exit /b 1
)

REM Vite's clean build deletes frontend/dist/.gitkeep; restore it so git status stays clean.
type nul > "frontend\dist\.gitkeep"

echo.
echo === Done ===
echo Output: build\bin\ssh-terminal.exe
echo The exe is portable. Copy the folder to any Windows machine.
echo Data files (settings.json, hosts.json, secret.key, known_hosts) are
echo created next to the exe in a "data" subfolder on first run.
endlocal
