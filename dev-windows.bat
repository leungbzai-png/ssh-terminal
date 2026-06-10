@echo off
setlocal

REM ============================================================
REM  SSH Terminal - Windows dev server
REM  Run from the project root directory.
REM  Requires: Go 1.22+, Node.js 18+, Wails CLI v2.12+
REM ============================================================

where go >nul 2>nul
if errorlevel 1 (
  echo [!] Go not found in PATH. Install Go 1.22+ and ensure it is on PATH.
  exit /b 1
)

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

wails dev
endlocal
