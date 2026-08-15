[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::InputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
Write-Host "Stopping Kwork Assistant daemon..."

$processes = Get-Process -Name "kwork-assistant" -ErrorAction SilentlyContinue

if ($processes) {
    foreach ($p in $processes) {
        Write-Host "Stopping process ID $($p.Id)..."
        Stop-Process -Id $p.Id -Force
    }
    Write-Host "Daemon stopped successfully." -ForegroundColor Green
} else {
    Write-Host "Kwork Assistant is not currently running." -ForegroundColor Yellow
}
