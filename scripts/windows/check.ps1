$AppDir = $PSScriptRoot

Write-Host "========================================"
Write-Host "       KWORK ASSISTANT CHECK            "
Write-Host "========================================"

$AllOk = $true

function Print-Status($Name, $Ok) {
    if ($Ok) {
        Write-Host "$Name`: OK" -ForegroundColor Green
    } else {
        Write-Host "$Name`: FAIL" -ForegroundColor Red
        $global:AllOk = $false
    }
}

$ExeFile = Join-Path $AppDir "kwork-assistant.exe"
$HasExe = Test-Path $ExeFile
Print-Status "Executable" $HasExe

$EnvFile = Join-Path $AppDir ".env"
$HasEnv = Test-Path $EnvFile
Print-Status ".env" $HasEnv

$DataDir = Join-Path $AppDir "data"
$HasData = Test-Path $DataDir
Print-Status "Data directory" $HasData

$LogsDir = Join-Path $AppDir "logs"
$HasLogs = Test-Path $LogsDir
Print-Status "Logs directory" $HasLogs

Write-Host "`n"

$EnvVars = @("TELEGRAM_BOT_TOKEN", "TELEGRAM_OWNER_CHAT_ID", "KWORK_LOGIN", "KWORK_PASSWORD", "KWORK_PHONE_LAST")
foreach ($Var in $EnvVars) {
    $IsSet = $false
    if ($HasEnv) {
        $Lines = Get-Content $EnvFile
        foreach ($Line in $Lines) {
            if ($Line -match "^$Var=(.+)$") {
                $Val = $matches[1].Trim()
                if ($Val -ne "") {
                    $IsSet = $true
                }
            }
        }
    }
    
    if ($IsSet) {
        Write-Host "$Var: SET" -ForegroundColor Green
    } else {
        Write-Host "$Var: MISSING" -ForegroundColor Red
        $AllOk = $false
    }
}

Write-Host "`n"

$OllamaCmd = Get-Command "ollama" -ErrorAction SilentlyContinue
$HasOllamaCmd = [bool]$OllamaCmd
Print-Status "Ollama" $HasOllamaCmd

$HasModel = $false
if ($HasOllamaCmd) {
    $ModelList = ollama list
    if ($ModelList -match "gemma4:e4b") {
        $HasModel = $true
    }
}
Print-Status "Model gemma4:e4b" $HasModel

if ($HasExe -and $HasEnv) {
    & $ExeFile health | Out-Null
    $AppOk = ($LASTEXITCODE -eq 0)
    Print-Status "Application" $AppOk

    & $ExeFile kwork-health | Out-Null
    $KworkOk = ($LASTEXITCODE -eq 0)
    Print-Status "Kwork auth" $KworkOk

    & $ExeFile telegram-health | Out-Null
    $TgOk = ($LASTEXITCODE -eq 0)
    Print-Status "Telegram API" $TgOk
} else {
    Print-Status "Application" $false
    Print-Status "Kwork auth" $false
    Print-Status "Telegram API" $false
}

Write-Host "`n"

if (-not $AllOk) {
    exit 1
}
exit 0
