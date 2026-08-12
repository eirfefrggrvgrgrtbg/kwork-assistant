# Kwork Assistant - Stage 1 (Local Core)

This is a local AI assistant for finding and evaluating freelance orders on Kwork. 
This repository represents STAGE 1: The robust local application core using Go, SQLite, and Ollama.

## Architecture
- **Go**: Primary application logic.
- **SQLite**: Local database for storing application metadata and AI test run history.
- **Ollama**: Local AI runtime using Gemma 4 model (`gemma4:e4b`).

## macOS Setup Instructions

1. **Verify Go is installed**
   ```bash
   go version
   ```

2. **Install and Start Ollama**
   Download from [ollama.com](https://ollama.com/download) or install via Homebrew:
   ```bash
   brew install ollama
   ```
   *Make sure the Ollama application is running in the background.*

3. **Download the expected AI Model**
   The application requires the Gemma 4 model (`gemma4:e4b`). Run this command manually to see the download progress:
   ```bash
   ollama run gemma4:e4b
   ```
   You can exit the interactive prompt (`/bye`) once it's downloaded.

4. **Verify Setup**
   Run the macOS setup script to verify dependencies:
   ```bash
   ./scripts/setup_mac.sh
   ```

5. **Configuration**
   Create your environment file:
   ```bash
   cp .env.example .env
   ```

## Running the Application

### Health Check
Check the status of the app, SQLite database, Ollama service, and the selected model:
```bash
go run ./cmd/app health
```

### Run AI Test
Send a predefined freelance job evaluation prompt to the local AI model. The structured response will be parsed and saved to the SQLite database:
```bash
go run ./cmd/app ai-test
```
