package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
)

type NotificationChannel int

const (
	Email NotificationChannel = iota
	SMS
	Push
)

type NotificationStatus int

const (
	Pending NotificationStatus = iota
	Sent
	Failed
)

type Notification struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	Channel          NotificationChannel
	RecipientAddress string
	Subject          string
	Body             string
	Status           NotificationStatus
	FailureReason    string
	CreatedAt        time.Time
	SentAt           *time.Time
}

type NotificationRepository interface {
	NextID() (uuid.UUID, error)
	Create(notification *Notification) error
	Update(notification *Notification) error
}

type NotificationSender interface {
	Send(recipient, subject, body string) error
}
