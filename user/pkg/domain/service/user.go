package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"user/pkg/domain/model"
)

var (
	ErrPasswordTooShort    = errors.New("password is too short")
	ErrUserCannotBeChanged = errors.New("user cannot be changed in its current state")
)

const minPasswordLength = 8

type Event interface {
	Type() string
}

type EventDispatcher interface {
	Dispatch(event Event) error
}

type UserService interface {
	RegisterNewUser(firstName, lastName, email, plainTextPassword string) (*model.User, error)
	UpdateUserProfile(userID uuid.UUID, firstName, lastName string) error
	SuspendUser(userID uuid.UUID) error
	ActivateUser(userID uuid.UUID) error
	DeactivateUser(userID uuid.UUID) error
}

func NewUserService(repo model.UserRepository, passManager model.PasswordManager, dispatcher EventDispatcher) UserService {
	return &userService{
		repo:        repo,
		passManager: passManager,
		dispatcher:  dispatcher,
	}
}

type userService struct {
	repo        model.UserRepository
	passManager model.PasswordManager
	dispatcher  EventDispatcher
}

func (s *userService) RegisterNewUser(firstName, lastName, email, plainTextPassword string) (*model.User, error) {
	if len(plainTextPassword) < minPasswordLength {
		return nil, ErrPasswordTooShort
	}

	if _, err := s.repo.FindByEmail(email); err == nil {
		return nil, model.ErrEmailTaken
	}

	hashedPassword, err := s.passManager.Hash(plainTextPassword)
	if err != nil {
		return nil, err
	}

	userID, err := s.repo.NextID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := &model.User{
		ID:             userID,
		Email:          email,
		HashedPassword: hashedPassword,
		FirstName:      firstName,
		LastName:       lastName,
		Status:         model.Active,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	_ = s.dispatcher.Dispatch(model.UserRegistered{
		UserID:    userID,
		Email:     email,
		FirstName: firstName,
	})

	return user, nil
}

func (s *userService) UpdateUserProfile(userID uuid.UUID, firstName, lastName string) error {
	user, err := s.repo.Find(userID)
	if err != nil {
		return err
	}

	if user.Status == model.Deactivated {
		return ErrUserCannotBeChanged
	}

	user.FirstName = firstName
	user.LastName = lastName
	user.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(user); err != nil {
		return err
	}

	_ = s.dispatcher.Dispatch(model.UserProfileUpdated{UserID: userID})
	return nil
}

func (s *userService) SuspendUser(userID uuid.UUID) error {
	return s.changeStatus(userID, model.Suspended)
}

func (s *userService) ActivateUser(userID uuid.UUID) error {
	return s.changeStatus(userID, model.Active)
}

func (s *userService) DeactivateUser(userID uuid.UUID) error {
	return s.changeStatus(userID, model.Deactivated)
}

func (s *userService) changeStatus(userID uuid.UUID, newStatus model.UserStatus) error {
	user, err := s.repo.Find(userID)
	if err != nil {
		return err
	}

	oldStatus := user.Status
	if oldStatus == newStatus || oldStatus == model.Deactivated {
		return nil
	}

	user.Status = newStatus
	user.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(user); err != nil {
		return err
	}

	_ = s.dispatcher.Dispatch(model.UserStatusChanged{
		UserID:    userID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
	})

	if newStatus == model.Deactivated {
		_ = s.dispatcher.Dispatch(model.UserDeactivated{UserID: userID})
	}

	return nil
}
