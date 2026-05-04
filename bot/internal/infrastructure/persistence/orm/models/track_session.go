package models

type TrackSessionModel struct {
	ChatID int64  `gorm:"column:chat_id;primaryKey"`
	State  string `gorm:"column:state;not null"`
	URL    string `gorm:"column:url;not null"`
}

func (TrackSessionModel) TableName() string {
	return "track_sessions"
}
