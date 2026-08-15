$ErrorActionPreference = "Stop"

$AppDir = $PSScriptRoot

Write-Host "Setting up Kwork Assistant..."
Write-Host "Application Directory: $AppDir"

# 1. Create directories
$DataDir = Join-Path $AppDir "data"
$LogsDir = Join-Path $AppDir "logs"

if (-not (Test-Path $DataDir)) {
    New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
    Write-Host "Created data directory."
}
if (-not (Test-Path $LogsDir)) {
    New-Item -ItemType Directory -Force -Path $LogsDir | Out-Null
    Write-Host "Created logs directory."
}

# 2. Setup .env
$EnvFile = Join-Path $AppDir ".env"
$EnvExample = Join-Path $AppDir ".env.example"

if (-not (Test-Path $EnvFile)) {
    if (Test-Path $EnvExample) {
        Copy-Item -Path $EnvExample -Destination $EnvFile
        Write-Host "Copied .env.example to .env. Please fill in your credentials in .env file."
    } else {
        Write-Warning "No .env.example found."
    }
} else {
    Write-Host ".env file already exists."
}

# 3. Check for executable
$ExeFile = Join-Path $AppDir "kwork-assistant.exe"
if (-not (Test-Path $ExeFile)) {
    Write-Error "kwork-assistant.exe not found in $AppDir. Please ensure you extracted the full release."
    exit 1
} else {
    Write-Host "Found kwork-assistant.exe"
}

# 4. Check Ollama
$OllamaExists = Get-Command "ollama" -ErrorAction SilentlyContinue
if (-not $OllamaExists) {
    Write-Host "==========================================================" -ForegroundColor Red
    Write-Host "OLLAMA IS NOT INSTALLED OR NOT IN PATH" -ForegroundColor Red
    Write-Host "Kwork Assistant requires Ollama to run the AI."
    Write-Host "Please download and install Ollama from: https://ollama.com"
    Write-Host "After installation, run this setup script again."
    Write-Host "==========================================================" -ForegroundColor Red
    exit 1
}

Write-Host "Ollama command found."

# 5. Check Model
$ModelName = "gemma4:e4b"
$ModelList = ollama list
if ($ModelList -match $ModelName) {
    Write-Host "Model $ModelName is already installed."
} else {
    Write-Host "Model $ModelName is not installed. Pulling it now (this may take a while)..."
    ollama pull $ModelName
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to pull model $ModelName. Please check your internet connection or run 'ollama pull $ModelName' manually."
        exit 1
    }
    Write-Host "Model $ModelName installed successfully."
}

Write-Host "Setup complete. Please fill your .env file and then run start.ps1" -ForegroundColor Green
