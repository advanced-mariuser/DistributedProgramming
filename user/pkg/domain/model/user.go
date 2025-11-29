package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email is already taken")
)

type UserStatus int

const (
	PendingVerification UserStatus = iota
	Active
	Suspended
	Deactivated
)

type User struct {
	ID             uuid.UUID
	Email          string
	HashedPassword string
	FirstName      string
	LastName       string
	Status         UserStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserRepository interface {
	NextID() (uuid.UUID, error)
	Create(user *User) error
	Update(user *User) error
	Find(id uuid.UUID) (*User, error)
	FindByEmail(email string) (*User, error)
}

type PasswordManager interface {
	Hash(plainTextPassword string) (string, error)
	Check(hashedPassword, plainTextPassword string) (bool, error)
}
