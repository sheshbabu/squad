# Database Guidelines

## Stack
- SQLite with the FTS5 extension, global `sqlite.DB` from `commons/sqlite/db.go`
- WAL with `_busy_timeout` and `_txlock=immediate`, because workers, the scheduler and handlers write concurrently
- All Go commands must include `--tags "fts5"`

## Schema & Migrations
- Migrations are sequential SQL files in `./migrations/` with format `<version>_<title>.sql`
- Main tables: users, sessions, bots, threads, messages, attachments, memories, schedules, connections, jobs, and `messages_fts` (kept in sync by triggers, user and assistant messages only)
- Timestamps compared in SQL are stored in UTC in SQLite's `YYYY-MM-DD HH:MM:SS` format so they compare against `CURRENT_TIMESTAMP`

## Query Patterns
- Always use parameterized queries with `?` placeholders
- Defer `rows.Close()` for multi-row queries
- Use transactions for multi-step operations with defer rollback pattern (except `commons/sqlite/migrate.go`, which is startup-only and rolls back manually)
- A domain write and the job it triggers share one transaction (`queue.EnqueueTx`), then call `queue.Wake()` after commit
- Model files (`*_model.go`) contain only database operations, and all feature SQL lives in them. Handlers and background code call model functions, including ones that run a transaction. Only `commons/` packages and `main.go` touch `sqlite.DB` directly
- Handle `sql.ErrNoRows` separately using `errors.Is()`
