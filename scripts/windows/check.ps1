$AppDir = $PSScriptRoot

Write-Host "============================================="
Write-Host "   Kwork Assistant Windows Check             "
Write-Host "============================================="

function Print-Status($Name, $Ok) {
    if ($Ok) {
        Write-Host "[OK]   $Name" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] $Name" -ForegroundColor Red
    }
}

# 1. Executable
$ExeFile = Join-Path $AppDir "kwork-assistant.exe"
$HasExe = Test-Path $ExeFile
Print-Status "Executable" $HasExe

# 2. .env
$EnvFile = Join-Path $AppDir ".env"
$HasEnv = Test-Path $EnvFile
Print-Status ".env" $HasEnv

# 3. data directory
$DataDir = Join-Path $AppDir "data"
$HasData = Test-Path $DataDir
Print-Status "data directory" $HasData

# 4. Ollama command
$OllamaCmd = Get-Command "ollama" -ErrorAction SilentlyContinue
$HasOllamaCmd = [bool]$OllamaCmd
Print-Status "Ollama command" $HasOllamaCmd

# 5. Ollama API
$HasOllamaApi = $false
try {
    $response = Invoke-RestMethod -Uri "http://127.0.0.1:11434/api/tags" -Method Get -ErrorAction Stop
    $HasOllamaApi = $true
} catch {
    $HasOllamaApi = $false
}
Print-Status "Ollama API" $HasOllamaApi

# 6. gemma4:e4b
$HasModel = $false
if ($HasOllamaApi) {
    if ($response.models.name -contains "gemma4:e4b") {
        $HasModel = $true
    }
}
Print-Status "gemma4:e4b" $HasModel

# 7. Check ENV variables manually
Write-Host "`nEnvironment Variables:"
$EnvVars = @("KWORK_LOGIN", "KWORK_PASSWORD", "TELEGRAM_BOT_TOKEN", "TELEGRAM_OWNER_CHAT_ID")
foreach ($Var in $EnvVars) {
    # Try to load it from .env or current env
    # Note: A real parse would be better, but we can just regex the file
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
        Write-Host "$Var: SET"
    } else {
        Write-Host "$Var: MISSING" -ForegroundColor Red
    }
}

Write-Host "`nRunning internal health checks..."
if ($HasExe) {
    Write-Host "`n--- App Health ---"
    & $ExeFile health
    $AppOk = ($LASTEXITCODE -eq 0)
    Print-Status "Application health" $AppOk

    Write-Host "`n--- Kwork Auth ---"
    & $ExeFile kwork-health
    $KworkOk = ($LASTEXITCODE -eq 0)
    Print-Status "Kwork auth" $KworkOk

    Write-Host "`n--- Telegram API ---"
    & $ExeFile telegram-health
    $TgOk = ($LASTEXITCODE -eq 0)
    Print-Status "Telegram API" $TgOk
} else {
    Print-Status "Application health" $false
    Print-Status "Kwork auth" $false
    Print-Status "Telegram API" $false
}
