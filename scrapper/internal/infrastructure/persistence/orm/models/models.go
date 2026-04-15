package models

import "time"

type ChatModel struct {
	ID int64 `gorm:"column:id,primaryKey"`
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
	ID   int64  `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name;uniqueIndex;not null"`
}

func (TagModel) TableName() string {
	return "tags"
}

type LinkTagModel struct {
	LinkID int64 `gorm:"column:link_id;primaryKey"`
	TagID  int64 `gorm:"column:tag_id;primaryKey"`
}

func (LinkTagModel) TableName() string {
	return "link_tags"
}
