package model

import "github.com/google/uuid"

type UserRegistered struct {
	UserID    uuid.UUID
	Email     string
	FirstName string
}

func (e UserRegistered) Type() string { return "UserRegistered" }

type UserProfileUpdated struct {
	UserID uuid.UUID
}

func (e UserProfileUpdated) Type() string { return "UserProfileUpdated" }

type UserStatusChanged struct {
	UserID    uuid.UUID
	OldStatus UserStatus
	NewStatus UserStatus
}

func (e UserStatusChanged) Type() string { return "UserStatusChanged" }

type UserDeactivated struct {
	UserID uuid.UUID
}

func (e UserDeactivated) Type() string { return "UserDeactivated" }
