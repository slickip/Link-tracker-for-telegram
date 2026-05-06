CREATE TABLE chats (
    id BIGINT PRIMARY KEY
);

CREATE TABLE links (
    id BIGSERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    last_updated TIMESTAMP
);

CREATE TABLE subscriptions (
    chat_id BIGINT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    link_id BIGINT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    PRIMARY KEY (chat_id, link_id)
);

CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    name TEXT NOT NULL,

    UNIQUE (chat_id, name),
    UNIQUE (id, chat_id)
);

CREATE TABLE subscription_tags (
    chat_id BIGINT NOT NULL,
    link_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,

    PRIMARY KEY (chat_id, link_id, tag_id),

    FOREIGN KEY (chat_id, link_id)
        REFERENCES subscriptions(chat_id, link_id)
        ON DELETE CASCADE,

    FOREIGN KEY (tag_id, chat_id)
        REFERENCES tags(id, chat_id)
        ON DELETE CASCADE
);

CREATE INDEX idx_subscriptions_link_id ON subscriptions(link_id);
CREATE INDEX idx_subscription_tags_tag_id ON subscription_tags(tag_id);