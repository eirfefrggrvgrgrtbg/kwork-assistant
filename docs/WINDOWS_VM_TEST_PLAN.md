# Windows VM Portable Release Test Plan

This test plan defines the final manual testing steps to be executed on a clean Windows VM to ensure the `KworkAssistant-Windows-x64.zip` release artifact is fully functional.

## Prerequisites
- A clean Windows 10/11 VM.
- No Go compiler installed.
- No Antigravity IDE or developer tools.
- A dummy Telegram Bot token.
- A dummy Kwork account.

## Test 1: Extraction & Layout Verification
1. Copy `KworkAssistant-Windows-x64.zip` to the VM desktop.
2. Right-click and Extract All to `C:\KworkAssistant`.
3. Open `C:\KworkAssistant` in File Explorer.
4. Verify the presence of `kwork-assistant.exe`, `setup.ps1`, `start.ps1`, `stop.ps1`, `check.ps1`, `.env.example`, and `README_WINDOWS.md`.
5. Ensure `data` and `logs` folders do not exist yet (or are empty).

## Test 2: Pre-Ollama Setup
1. Right-click `setup.ps1` -> Run with PowerShell.
2. The script should immediately halt and display red text warning that Ollama is not installed.
3. Verify that `data` and `logs` were created.
4. Verify `.env` was copied from `.env.example`.

## Test 3: Ollama Installation & Setup
1. Install Ollama from https://ollama.com.
2. Re-run `setup.ps1`.
3. The script should detect Ollama and begin pulling `gemma4:e4b`.
4. Wait for the pull to complete successfully.

## Test 4: Environment Configuration
1. Open `.env` in Notepad.
2. Add a test Telegram Bot Token and Chat ID.
3. Add a test Kwork login and password.
4. Leave `EMAIL_` settings blank to test the optional IMAP feature.
5. Save `.env`.

## Test 5: Check Script
1. Right-click `check.ps1` -> Run with PowerShell.
2. Verify all basic checks report `[OK]`.
3. Verify `App Health`, `Kwork Auth`, and `Telegram API` all pass internal tests and report `[OK]`.
4. Ensure no secret values (passwords, tokens) are printed to the console output.

## Test 6: Daemon Start
1. Right-click `start.ps1` -> Run with PowerShell.
2. Verify the console displays logs indicating:
   - "Email intake disabled (IMAP config missing)"
   - "SQLite health check passed"
   - "Ollama health check passed"
   - "Daemon loop started"
3. Wait 1 minute and verify Telegram bot receives a startup message or that polling occurs without crashes.
4. Send `/start` to the Telegram bot and verify a response.

## Test 7: Daemon Stop
1. Focus the console window running the daemon.
2. Press `Ctrl+C`.
3. Verify the script says "Daemon stopped" gracefully.
4. Run `stop.ps1` to ensure it reports that the assistant is not currently running.

## Test 8: Data Persistence Check
1. Open `C:\KworkAssistant\data\`.
2. Verify `kwork-assistant.db` exists and has a non-zero size.
3. Verify `kwork-assistant.db-shm` and `kwork-assistant.db-wal` behave normally (if WAL is enabled).

If all tests pass, the Windows Portable Release is certified as fully functional and ready for deployment.
