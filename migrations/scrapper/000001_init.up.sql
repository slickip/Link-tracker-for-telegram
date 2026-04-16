CREATE TABLE chats (
    id BIGINT PRIMARY KEY
);

CREATE TABLE links (
    id BIGSERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    last_updated TIMESTAMP
);

CREATE TABLE subscriptions (
    chat_id BIGINT REFERENCES chats(id) ON DELETE CASCADE,
    link_id BIGINT REFERENCES links(id) ON DELETE CASCADE,
    PRIMARY KEY (chat_id, link_id)
);

CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,
    name TEXT UNIQUE
);

CREATE TABLE link_tags (
    link_id BIGINT REFERENCES links(id) ON DELETE CASCADE,
    tag_id BIGINT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (link_id, tag_id)
);

CREATE INDEX idx_links_url ON links(url);
CREATE INDEX idx_subscriptions_chat_id ON subscriptions(chat_id);
CREATE INDEX idx_subscriptions_link_id ON subscriptions(link_id);