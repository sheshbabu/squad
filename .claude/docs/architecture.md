# Architecture

## Layout
- **Entry Point**: `main.go` sets up routes, the queue workers, the scheduler and background tasks
- **Feature-based Structure**: each feature has its own directory under `features/` with handlers, models and components
  - `users/` - Login and onboarding for the single user
  - `bots/` - Bot CRUD and the bot page layout (sidebar, list column, detail column)
  - `gateways/` - Model and provider pickers
  - `threads/` - Threads, messages, attachments, the SSE endpoint and the chat UI
  - `agent/` - The agent loop, context building and thread titles
  - `tools/` - Tools bots can call (memory, files, fetch, Exa, Hacker News, Reddit, Zen, Calendar)
  - `connections/` - Per-bot secrets that unlock tool groups, like Zen API tokens
  - `memories/`, `files/`, `schedules/`, `search/`
- **Commons**: `sqlite/`, `utils/`, `session/`, `auth/`, `queue/` (durable job queue), `llm/` (gateways and stream client), `stream/` (SSE hub), plus frontend `components/`, `contexts/`, `http/ApiClient.js`

## How a reply happens
1. Sending a message inserts it, sets the thread to `working` and enqueues an `agent_run` job in one transaction (`threads.StartRunTx`).
2. A queue worker claims the job and runs `agent.runThread`, which reloads the thread from the database before every step: run missing tool results, else call the model, else finish.
3. Model text streams through `stream.AppendPartial` to browsers subscribed on `GET /api/threads/{id}/events/`. Every completed step is saved before the next one starts.
4. The database is the source of truth. A retry or a restart resumes from the last saved step, and browsers refetch the thread whenever their stream (re)opens.

## Durable queue (`commons/queue/`)
- `jobs` table, claimed atomically with `UPDATE ... RETURNING`, run by 2 worker goroutines woken by `Wake()` or a 2s ticker
- Errors retry with backoff (5s, 15s, 45s... capped at 5m) unless wrapped with `queue.Permanent`, then `OnFailed` runs
- Jobs left `running` at startup are requeued, since only this process runs jobs
- `job_key` lets `queue.Cancel` stop a running job (the Stop button) or delete a queued one

## Gateways (`commons/llm/`)
Any OpenAI-compatible chat completions API. OpenRouter is registered from `OPENROUTER_API_KEY` and adds provider pinning (`provider.order` with `allow_fallbacks: false`) and usage with cost. Assistant messages keep `reasoning_details` and send them back, since reasoning models need them across tool calls.

## Scheduled tasks
The scheduler checks every 30s for due schedules, then creates a new thread with the schedule's prompt and starts a run. `next_run_at` is computed in local time (`TZ`). Missed runs fire once on startup. "Run now" starts a run without moving `next_run_at`.

## Environment Variables
- `DEV_MODE=true` - Serve assets from disk
- `PORT` - Server port (default: 8080)
- `DATA_FOLDER` - Database, bot folders and attachments (default: `./data`)
- `TZ` - Time zone for schedules and the date in prompts
- `OPENROUTER_API_KEY` - The OpenRouter gateway
- `EXA_API_KEY`, `REDDIT_CLIENT_ID`, `REDDIT_CLIENT_SECRET`, `REDDIT_USERNAME`, `REDDIT_PASSWORD`, `REDDIT_USER_AGENT`, `ZEN_URL` - Tools, each hidden until its keys are set. Each bot can turn tool groups off, and groups like Zen need a connection (name plus secret) on the bot, which can have several

## Background Tasks
- Queue workers (continuous)
- Scheduler (30 seconds)
- Session cleanup (24 hours)
