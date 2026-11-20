CREATE TABLE connections (
    connection_id INTEGER PRIMARY KEY AUTOINCREMENT,
    bot_id        INTEGER NOT NULL REFERENCES bots(bot_id) ON DELETE CASCADE,
    group_id      TEXT NOT NULL,
    name          TEXT NOT NULL,
    secret        TEXT NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_connections_name ON connections (bot_id, group_id, name);

INSERT INTO connections (bot_id, group_id, name, secret)
SELECT bot_id, 'zen', name, zen_api_token FROM bots WHERE zen_api_token != '';

ALTER TABLE bots DROP COLUMN zen_api_token;
