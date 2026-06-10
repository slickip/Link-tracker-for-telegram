package domain

import "time"

type OutboxStatus string

const (
	OutboxStatusPending OutboxStatus = "PENDING"
	OutboxStatusSent    OutboxStatus = "SENT"
	OutboxStatusFailed  OutboxStatus = "FAILED"
)

type OutboxMessage struct {
	ID         int64
	Topic      string
	MessageKey string
	Payload    []byte
	Status     OutboxStatus
	Attempts   int
	LastError  *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	SentAt     *time.Time
}
