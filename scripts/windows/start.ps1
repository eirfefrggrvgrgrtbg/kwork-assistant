$AppDir = $PSScriptRoot

$EnvFile = Join-Path $AppDir ".env"
$ConfigOk = $false

if (Test-Path $EnvFile) {
    $Lines = Get-Content $EnvFile
    $HasTg = $false
    $HasKwork = $false
    foreach ($Line in $Lines) {
        if ($Line -match "^TELEGRAM_BOT_TOKEN=(.+)$" -and $matches[1].Trim() -ne "") {
            $HasTg = $true
        }
        if ($Line -match "^KWORK_LOGIN=(.+)$" -and $matches[1].Trim() -ne "") {
            $HasKwork = $true
        }
    }
    if ($HasTg -and $HasKwork) {
        $ConfigOk = $true
    }
}

if (-not $ConfigOk) {
    Write-Host "Configuration incomplete." -ForegroundColor Red
    Write-Host "Run:" -ForegroundColor Red
    Write-Host ".\setup.ps1" -ForegroundColor Yellow
    exit 1
}

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
    & $ExeFile daemon
} catch {
    Write-Host "`nDaemon stopped."
}
