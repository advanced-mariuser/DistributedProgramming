package model

import "github.com/google/uuid"

type NotificationSent struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID
	Channel        NotificationChannel
}

func (e NotificationSent) Type() string { return "NotificationSent" }

type NotificationFailed struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID
	Channel        NotificationChannel
	Reason         string
}

func (e NotificationFailed) Type() string { return "NotificationFailed" }
