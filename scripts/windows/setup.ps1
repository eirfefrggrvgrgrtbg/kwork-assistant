$ErrorActionPreference = "Stop"

$AppDir = $PSScriptRoot

# Helper: Set-EnvValue
function Set-EnvValue {
    param(
        [string]$Key,
        [string]$Value
    )
    $EnvFile = Join-Path $AppDir ".env"
    if (-not (Test-Path $EnvFile)) {
        New-Item -Path $EnvFile -ItemType File -Force | Out-Null
    }
    
    $lines = Get-Content $EnvFile
    $found = $false
    $newLines = @()
    
    foreach ($line in $lines) {
        if ($line -match "^$Key=(.*)$") {
            $newLines += "$Key=$Value"
            $found = $true
        } else {
            $newLines += $line
        }
    }
    
    if (-not $found) {
        $newLines += "$Key=$Value"
    }
    
    [System.IO.File]::WriteAllLines($EnvFile, $newLines)
    Write-Host "$Key: SET" -ForegroundColor Green
}

# Helper: Test-TelegramToken
function Test-TelegramToken($Token) {
    try {
        $url = "https://api.telegram.org/bot$Token/getMe"
        $response = Invoke-RestMethod -Uri $url -Method Get -ErrorAction Stop
        if ($response.ok) {
            return $response.result
        }
    } catch {
        return $null
    }
    return $null
}

function Get-TelegramOwner($Token) {
    $baselineId = 0
    try {
        $Updates = Invoke-RestMethod -Uri "https://api.telegram.org/bot$Token/getUpdates?offset=-1&limit=1&timeout=0" -ErrorAction Stop
        if ($Updates.ok -and $Updates.result.Count -gt 0) {
            $baselineId = $Updates.result[-1].update_id
        }
    } catch {
        # Ignore initial error, baseline is 0
    }

    Write-Host "----------------------------------------" -ForegroundColor Cyan
    Write-Host "Теперь откройте Telegram." -ForegroundColor Cyan
    Write-Host "1. Найдите своего нового бота" -ForegroundColor Cyan
    Write-Host "2. Нажмите START или отправьте /start" -ForegroundColor Cyan
    Write-Host "3. После этого вернитесь сюда и нажмите Enter." -ForegroundColor Cyan
    Write-Host "----------------------------------------" -ForegroundColor Cyan
    Read-Host "Press ENTER when done..." | Out-Null

    Write-Host "Waiting for /start message from you (timeout 60s)..."
    
    $timeout = 60
    $startTime = Get-Date
    $offset = $baselineId + 1

    while (((Get-Date) - $startTime).TotalSeconds -lt $timeout) {
        Start-Sleep -Seconds 2
        try {
            $url = "https://api.telegram.org/bot$Token/getUpdates?offset=$offset&timeout=5"
            $Resp = Invoke-RestMethod -Uri $url -Method Get -ErrorAction Stop
            if ($Resp.ok -and $Resp.result) {
                foreach ($upd in $Resp.result) {
                    $offset = [Math]::Max($offset, $upd.update_id + 1)
                    if ($upd.message -and $upd.message.chat.type -eq "private" -and $upd.message.text -match "^/start(\s|$)") {
                        $from = $upd.message.from
                        $chatId = $upd.message.chat.id
                        
                        Write-Host "`nTelegram account found:"
                        Write-Host "Name: $($from.first_name) $($from.last_name)"
                        Write-Host "Username: @$($from.username)"
                        Write-Host "Chat ID: $chatId"
                        
                        $ans = Read-Host "Is this your account? [Y/N]"
                        if ($ans -match "^[Yy]") {
                            return $chatId
                        } else {
                            Write-Host "Continuing to wait..."
                        }
                    }
                }
            }
        } catch {
            $errMsg = $_.Exception.Message
            if ($errMsg -match "409" -or $errMsg -match "Conflict") {
                Write-Host "`nAnother instance of the bot may already be running." -ForegroundColor Red
                Write-Host "Stop Kwork Assistant and retry setup." -ForegroundColor Red
                return $null
            }
        }
    }
    Write-Host "`nTimeout reached." -ForegroundColor Yellow
    return $null
}

Write-Host "========================================"
Write-Host "       KWORK ASSISTANT SETUP            "
Write-Host "========================================"

# ==================================================
# [1/5] Directories
# ==================================================
Write-Host "`n[1/5] Directories" -ForegroundColor Yellow

$ExeFile = Join-Path $AppDir "kwork-assistant.exe"
if (-not (Test-Path $ExeFile)) {
    Write-Host "[FAIL] kwork-assistant.exe not found" -ForegroundColor Red
    exit 1
}

$DataDir = Join-Path $AppDir "data"
$LogsDir = Join-Path $AppDir "logs"

if (-not (Test-Path $DataDir)) {
    New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
}
if (-not (Test-Path $LogsDir)) {
    New-Item -ItemType Directory -Force -Path $LogsDir | Out-Null
}
Write-Host "Directories checked."

$EnvFile = Join-Path $AppDir ".env"
$EnvExample = Join-Path $AppDir ".env.example"

$UpdateConfig = $true

if (-not (Test-Path $EnvFile)) {
    if (Test-Path $EnvExample) {
        Copy-Item -Path $EnvExample -Destination $EnvFile
        Write-Host "Copied .env.example to .env."
    }
} else {
    $ans = Read-Host "Existing configuration found. Update configuration? [Y/N]"
    if ($ans -notmatch "^[Yy]") {
        Write-Host "Keeping existing configuration and continuing."
        $UpdateConfig = $false
    }
}

# Apply default runtime settings explicitly to ensure they exist
Set-EnvValue "KWORK_PROJECT_WATCH_ENABLED" "true"
Set-EnvValue "KWORK_PROJECT_POLL_INTERVAL" "60s"
Set-EnvValue "KWORK_SUITABLE_SCORE" "80"
Set-EnvValue "KWORK_CHAT_SYNC_ENABLED" "true"

if ($UpdateConfig) {
    # ==================================================
    # [2/5] Telegram Bot
    # ==================================================
    Write-Host "`n[2/5] Telegram Bot" -ForegroundColor Yellow
    Write-Host "Введите Telegram Bot Token, полученный от @BotFather:"
    $SecureToken = Read-Host -AsSecureString
    $BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($SecureToken)
    $Token = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)
    [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($BSTR)

    $BotInfo = Test-TelegramToken $Token
    if (-not $BotInfo) {
        Write-Host "[FAIL] Telegram rejected this bot token." -ForegroundColor Red
        $retry = Read-Host "Retry? [Y/N]"
        if ($retry -match "^[Yy]") {
            Write-Host "Please run setup.ps1 again." -ForegroundColor Yellow
        }
        exit 1
    }

    Write-Host "[OK] Telegram bot connected" -ForegroundColor Green
    Write-Host "Bot username: @$($BotInfo.username)"

    Set-EnvValue "TELEGRAM_BOT_TOKEN" $Token

    $OwnerChatId = Get-TelegramOwner $Token
    if ($OwnerChatId) {
        Set-EnvValue "TELEGRAM_OWNER_CHAT_ID" $OwnerChatId
    } else {
        Write-Host "Failed to get Owner Chat ID. Setup incomplete." -ForegroundColor Red
        exit 1
    }

    # ==================================================
    # [3/5] Kwork Account
    # ==================================================
    Write-Host "`n[3/5] Kwork Account" -ForegroundColor Yellow

    $KworkOkLoop = $false
    while (-not $KworkOkLoop) {
        $KworkLogin = Read-Host "Kwork login/email"

        Write-Host "Kwork password (input hidden):" -NoNewline
        $SecureKworkPass = Read-Host -AsSecureString
        $BSTRPass = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($SecureKworkPass)
        $KworkPass = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTRPass)
        [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($BSTRPass)

        $KworkPhone = Read-Host "Last phone digits (e.g. 1234)"

        Set-EnvValue "KWORK_LOGIN" $KworkLogin
        Set-EnvValue "KWORK_PASSWORD" $KworkPass
        Set-EnvValue "KWORK_PHONE_LAST" $KworkPhone

        Write-Host "Checking Kwork connection..."
        & $ExeFile kwork-health
        if ($LASTEXITCODE -ne 0) {
            Write-Host "[FAIL] Kwork authentication failed." -ForegroundColor Red
            $retry = Read-Host "Re-enter Kwork credentials? [Y/N]"
            if ($retry -notmatch "^[Yy]") {
                Write-Host "Setup incomplete. Exiting." -ForegroundColor Red
                exit 1
            }
        } else {
            Write-Host "[OK] Kwork authentication successful." -ForegroundColor Green
            $KworkOkLoop = $true
        }
    }
}

# ==================================================
# [4/5] Ollama
# ==================================================
Write-Host "`n[4/5] Ollama" -ForegroundColor Yellow
$OllamaExists = Get-Command "ollama" -ErrorAction SilentlyContinue
if (-not $OllamaExists) {
    Write-Host "[FAIL] Ollama is not installed." -ForegroundColor Red
    Write-Host "Please install Ollama from https://ollama.com and run setup.ps1 again."
    exit 1
}

# Check API
$OllamaRunning = $false
try {
    $apiResponse = Invoke-RestMethod -Uri "http://127.0.0.1:11434/api/tags" -Method Get -ErrorAction Stop
    $OllamaRunning = $true
} catch {
    $OllamaRunning = $false
}

if (-not $OllamaRunning) {
    Write-Host "[FAIL] Ollama is installed but not running." -ForegroundColor Red
    Write-Host "Start Ollama and run setup.ps1 again."
    exit 1
}

Write-Host "[OK] Ollama API is running." -ForegroundColor Green

$ModelName = "gemma4:e4b"
$ModelList = ollama list
if ($ModelList -match $ModelName) {
    Write-Host "[OK] Model $ModelName is already installed." -ForegroundColor Green
} else {
    Write-Host "Model $ModelName is not installed." -ForegroundColor Yellow
    $pull = Read-Host "Download it now? [Y/N]"
    if ($pull -match "^[Yy]") {
        Write-Host "Pulling model $ModelName (this may take a while)..."
        ollama pull $ModelName
        if ($LASTEXITCODE -ne 0) {
            Write-Host "[FAIL] Failed to pull model $ModelName." -ForegroundColor Red
            exit 1
        }
    } else {
        Write-Host "Setup incomplete. The AI flow requires the model to work." -ForegroundColor Red
        exit 1
    }
}

# ==================================================
# [5/5] Final Check
# ==================================================
Write-Host "`n[5/5] Final Check" -ForegroundColor Yellow

& $ExeFile health | Out-Null
$HealthOk = ($LASTEXITCODE -eq 0)

& $ExeFile kwork-health | Out-Null
$KworkOk = ($LASTEXITCODE -eq 0)

& $ExeFile telegram-health | Out-Null
$TgOk = ($LASTEXITCODE -eq 0)

Write-Host "`n========================================"
Write-Host "KWORK ASSISTANT SETUP RESULT"
Write-Host "========================================"

$AllOk = $true

function Print-And-Check($Name, $Ok) {
    if ($Ok) {
        Write-Host "[OK] $Name" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] $Name" -ForegroundColor Red
        $global:AllOk = $false
    }
}

Print-And-Check "Configuration" $true
Print-And-Check "SQLite" $HealthOk
Print-And-Check "Ollama" $HealthOk
Print-And-Check "gemma4:e4b" $HealthOk
Print-And-Check "Kwork" $KworkOk
Print-And-Check "Telegram" $TgOk
Print-And-Check "Owner Chat ID" $true

if ($AllOk) {
    Write-Host "`nSetup completed successfully." -ForegroundColor Green
    Write-Host "Next step:"
    Write-Host ".\start.ps1" -ForegroundColor Cyan
    Write-Host "========================================"
} else {
    Write-Host "`nSetup incomplete." -ForegroundColor Red
    Write-Host "Run .\check.ps1 for details." -ForegroundColor Yellow
    Write-Host "========================================"
    exit 1
}
