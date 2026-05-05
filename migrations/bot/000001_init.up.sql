CREATE TABLE track_sessions (
    chat_id BIGINT PRIMARY KEY,
    state TEXT NOT NULL,
    url TEXT NOT NULL DEFAULT ''
);