$AppDir = $PSScriptRoot

Write-Host "Running health checks before starting..."
& "$AppDir\check.ps1"

$ExeFile = Join-Path $AppDir "kwork-assistant.exe"
if (-not (Test-Path $ExeFile)) {
    Write-Error "kwork-assistant.exe not found."
    exit 1
}

Write-Host "`nStarting Kwork Assistant Daemon..." -ForegroundColor Green
Write-Host "Press Ctrl+C to stop the daemon gracefully."

try {
    # Run in current console so the user can see the logs
    & $ExeFile daemon
} catch {
    Write-Host "`nDaemon stopped."
}
