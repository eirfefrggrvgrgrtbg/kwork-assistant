# Mac Baseline Frozen

**Date**: 2026-08-15
**Tag**: `mac-baseline-v1`

## Environment
- **OS Architecture**: macOS
- **Go Version**: 1.22+ (as available in environment)
- **Ollama Model**: `gemma4:e4b`
- **Database Contract**: SQLite DB stored at `./data/kwork-assistant.db`. Explicit checks to not commit data files.

## Smoke Tests Passed
- **Security Check**: `.env` is fully ignored and untracked. No credentials leaked.
- **Automated Tests**: `go test ./...`, `go vet ./...`, `go build ./...` all pass.
- **Service Health**: App, SQLite, Ollama, Kwork Auth, and Telegram API all return OK.
- **Project Watcher Deduplication**: One-shot execution successfully fetches projects and avoids duplicate evaluation runs.
- **Bot Commands / UI**: Reply keyboard features `🔥 Заказы`, `💬 Диалоги`, etc. Pause/Resume commands toggle `auto_processing_enabled`.
- **Zero-Auto-Send Policy**: The AI does *not* automatically generate or send Kwork proposals until explicit manual action is taken.
- **Read-Only Integration**: Verified that Kwork writes are isolated; the bot only notifies and prepares drafts via Telegram.

## Known Limitations
- The project is fully Mac-specific right now and assumes Unix environment behavior (e.g., paths, background processes).

## Future Windows Port
- The Windows port MUST preserve this exact behavior, particularly the ZERO-AUTO-SEND policy and SQL table compatibility.
