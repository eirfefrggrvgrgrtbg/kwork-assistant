[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::InputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
$AppDir = $PSScriptRoot

$EnvFile = Join-Path $AppDir ".env"
$ConfigOk = $false

if (Test-Path $EnvFile) {
    $Lines = [System.IO.File]::ReadAllLines($EnvFile, [System.Text.Encoding]::UTF8)
    $HasTg = $false
    $HasTgOwner = $false
    $HasKwork = $false
    $HasKworkPass = $false
    $HasKworkPhone = $false
    
    foreach ($Line in $Lines) {
        if ($Line -match "^TELEGRAM_BOT_TOKEN=(.+)$" -and $matches[1].Trim() -ne "") { $HasTg = $true }
        if ($Line -match "^TELEGRAM_OWNER_CHAT_ID=(.+)$" -and $matches[1].Trim() -ne "") { $HasTgOwner = $true }
        if ($Line -match "^KWORK_LOGIN=(.+)$" -and $matches[1].Trim() -ne "") { $HasKwork = $true }
        if ($Line -match "^KWORK_PASSWORD=(.+)$" -and $matches[1].Trim() -ne "") { $HasKworkPass = $true }
        if ($Line -match "^KWORK_PHONE_LAST=(.+)$" -and $matches[1].Trim() -ne "") { $HasKworkPhone = $true }
    }
    if ($HasTg -and $HasTgOwner -and $HasKwork -and $HasKworkPass -and $HasKworkPhone) {
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
& "$AppDir