# Go Backend Guidelines

All Go commands must include the `--tags "fts5"` flag for SQLite FTS5 support.

## Error Handling
- Use error wrapping with context: `fmt.Errorf("context: %w", err)`
- Don't log errors you return. Log once where an error stops propagating: `SendErrorResponse` logs for handlers, and background loops, best-effort calls and swallowed errors log with `slog.Error()`
- Handle `sql.ErrNoRows` separately using `errors.Is()`
- Models return `utils.ErrNotFound` (wrapped with `%w`) when a lookup finds no row or an update/delete affects zero rows. Handlers pass 500 to `SendErrorResponse`, which sends 404 for `ErrNotFound`
- Job handlers wrap errors retrying can't fix with `queue.Permanent`
- Tool errors are returned to the model as text, never fail the run
- Use `panic()` only for critical initialization errors

## HTTP Handlers
- Function signature: `func HandleXXX(w http.ResponseWriter, r *http.Request)`
- Naming: `Handle{Action}{Resource}` (e.g., `HandleGetThreads`, `HandleSendMessage`)
- Structure: Parse/validate → Business logic → Response
- Send JSON with `utils.SendJSONResponse(w, status, value)`, never `json.NewEncoder(w)` directly
- Send errors with `utils.SendErrorResponse(w, status, code, message, err)`. Pass the error for failures so it gets logged, and `nil` for client mistakes

## API Endpoints
- RESTful patterns with a trailing slash: `GET /api/resource/`, `POST /api/resource/`, `PUT /api/resource/{id}/`
- Routes are registered in `main.go` with `addPublicRoute()` or `addSessionRoute()`
- There is a single user, so handlers never filter by user and nothing records who sent a message

## Struct Conventions
- Use one struct for DB and API unless their shapes differ; then name the DB struct `{Resource}Record`
- Request bodies that aren't a resource are `{Action}Request` (e.g., `SendMessageRequest`)
- Use camelCase JSON tags: `json:"fieldName"`
- Use `slog` with lowercase messages and key-value pairs

## Tests
- `commons/testutil.SetupDB(t)` gives a fresh database with every migration and a temp `DATA_FOLDER`
- Fake gateways and tool APIs with `httptest`, and register a fake gateway with `llm.RegisterGateway`
