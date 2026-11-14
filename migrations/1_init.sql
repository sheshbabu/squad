CREATE TABLE users (
    user_id       INTEGER PRIMARY KEY AUTOINCREMENT,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    expires_at TEXT
);

CREATE TABLE bots (
    bot_id        INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT NOT NULL,
    icon          TEXT NOT NULL DEFAULT '',
    color         TEXT NOT NULL DEFAULT 'gray',
    system_prompt TEXT NOT NULL DEFAULT '',
    gateway       TEXT NOT NULL DEFAULT 'openrouter',
    model         TEXT NOT NULL,
    provider      TEXT NOT NULL DEFAULT '',
    disabled_tool_groups TEXT NOT NULL DEFAULT '',
    zen_api_token TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE schedules (
    schedule_id INTEGER PRIMARY KEY AUTOINCREMENT,
    bot_id      INTEGER NOT NULL REFERENCES bots(bot_id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    prompt      TEXT NOT NULL,
    frequency   TEXT NOT NULL,
    time_of_day TEXT NOT NULL,
    weekdays    TEXT NOT NULL DEFAULT '',
    is_enabled  INTEGER NOT NULL DEFAULT 1,
    next_run_at TIMESTAMP NOT NULL,
    last_run_at TIMESTAMP,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_schedules_due ON schedules (is_enabled, next_run_at);

CREATE TABLE threads (
    thread_id     INTEGER PRIMARY KEY AUTOINCREMENT,
    bot_id        INTEGER NOT NULL REFERENCES bots(bot_id) ON DELETE CASCADE,
    schedule_id   INTEGER REFERENCES schedules(schedule_id) ON DELETE SET NULL,
    title         TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'idle',
    error_message TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_threads_bot ON threads (bot_id, updated_at);

CREATE TABLE messages (
    message_id        INTEGER PRIMARY KEY AUTOINCREMENT,
    thread_id         INTEGER NOT NULL REFERENCES threads(thread_id) ON DELETE CASCADE,
    role              TEXT NOT NULL,
    content           TEXT NOT NULL DEFAULT '',
    tool_calls        TEXT NOT NULL DEFAULT '',
    reasoning_details TEXT NOT NULL DEFAULT '',
    tool_call_id      TEXT NOT NULL DEFAULT '',
    tool_name         TEXT NOT NULL DEFAULT '',
    model             TEXT NOT NULL DEFAULT '',
    provider          TEXT NOT NULL DEFAULT '',
    prompt_tokens     INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    cost              REAL NOT NULL DEFAULT 0,
    is_cancelled      INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_thread ON messages (thread_id, message_id);

CREATE TABLE attachments (
    attachment_id INTEGER PRIMARY KEY AUTOINCREMENT,
    bot_id        INTEGER NOT NULL REFERENCES bots(bot_id) ON DELETE CASCADE,
    message_id    INTEGER REFERENCES messages(message_id) ON DELETE CASCADE,
    filename      TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    size          INTEGER NOT NULL,
    path          TEXT NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_attachments_message ON attachments (message_id);

CREATE TABLE memories (
    memory_id  INTEGER PRIMARY KEY AUTOINCREMENT,
    bot_id     INTEGER NOT NULL REFERENCES bots(bot_id) ON DELETE CASCADE,
    content    TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_memories_bot ON memories (bot_id);

CREATE TABLE jobs (
    job_id       INTEGER PRIMARY KEY AUTOINCREMENT,
    kind         TEXT NOT NULL,
    job_key      TEXT NOT NULL DEFAULT '',
    payload      TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'queued',
    attempts     INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    run_after    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_error   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_jobs_claim ON jobs (status, run_after);
CREATE INDEX idx_jobs_key ON jobs (job_key);

-- Only user and assistant text is searchable, tool output would drown the results.
CREATE VIRTUAL TABLE messages_fts USING fts5(content, content='messages', content_rowid='message_id');

CREATE TRIGGER messages_fts_insert AFTER INSERT ON messages WHEN new.role IN ('user', 'assistant') BEGIN
    INSERT INTO messages_fts (rowid, content) VALUES (new.message_id, new.content);
END;

CREATE TRIGGER messages_fts_delete AFTER DELETE ON messages WHEN old.role IN ('user', 'assistant') BEGIN
    INSERT INTO messages_fts (messages_fts, rowid, content) VALUES ('delete', old.message_id, old.content);
END;
