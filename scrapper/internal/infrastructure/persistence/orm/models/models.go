package models

import (
	"time"

	"gorm.io/datatypes"
)

type ChatModel struct {
	ID int64 `gorm:"column:id;primaryKey"`
}

func (ChatModel) TableName() string {
	return "chats"
}

type LinkModel struct {
	ID          int64     `gorm:"column:id;primaryKey"`
	URL         string    `gorm:"column:url;uniqueIndex;not null"`
	LastUpdated time.Time `gorm:"column:last_updated"`
}

func (LinkModel) TableName() string {
	return "links"
}

type SubscriptionModel struct {
	ChatID int64 `gorm:"column:chat_id;primaryKey"`
	LinkID int64 `gorm:"column:link_id;primaryKey"`
}

func (SubscriptionModel) TableName() string {
	return "subscriptions"
}

type TagModel struct {
	ID     int64  `gorm:"column:id;primaryKey"`
	ChatID int64  `gorm:"column:chat_id;not null;uniqueIndex:idx_tags_chat_name"`
	Name   string `gorm:"column:name;not null;uniqueIndex:idx_tags_chat_name"`
}

func (TagModel) TableName() string {
	return "tags"
}

type SubscriptionTagModel struct {
	ChatID int64 `gorm:"column:chat_id;primaryKey"`
	LinkID int64 `gorm:"column:link_id;primaryKey"`
	TagID  int64 `gorm:"column:tag_id;primaryKey"`
}

func (SubscriptionTagModel) TableName() string {
	return "subscription_tags"
}

type OutboxMessageModel struct {
	ID         int64          `gorm:"column:id;primaryKey"`
	Topic      string         `gorm:"column:topic;not null"`
	MessageKey string         `gorm:"column:message_key;not null"`
	Payload    datatypes.JSON `gorm:"column:payload;type:jsonb;not null"`
	Status     string         `gorm:"column:status;not null"`
	Attempts   int            `gorm:"column:attempts;not null"`
	LastError  *string        `gorm:"column:last_error"`
	CreatedAt  time.Time      `gorm:"column:created_at;not null"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;not null"`
	SentAt     *time.Time     `gorm:"column:sent_at"`
}

func (OutboxMessageModel) TableName() string {
	return "outbox_messages"
}
