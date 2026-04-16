DROP INDEX IF EXISTS idx_subscription_tags_tag_id;
DROP INDEX IF EXISTS idx_subscription_tags_link_id;
DROP INDEX IF EXISTS idx_subscription_tags_chat_id;
DROP INDEX IF EXISTS idx_subscriptions_link_id;
DROP INDEX IF EXISTS idx_subscriptions_chat_id;
DROP INDEX IF EXISTS idx_links_url;

DROP TABLE IF EXISTS subscription_tags;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS links;
DROP TABLE IF EXISTS chats;