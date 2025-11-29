package tests

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"user/pkg/domain/model"
	"user/pkg/domain/service"
)

func setup(t *testing.T) (service.UserService, *mockUserRepository, *mockPasswordManager, *mockEventDispatcher) {
	repo := &mockUserRepository{store: make(map[uuid.UUID]*model.User)}
	passManager := &mockPasswordManager{}
	dispatcher := &mockEventDispatcher{}
	userService := service.NewUserService(repo, passManager, dispatcher)
	return userService, repo, passManager, dispatcher
}

func TestRegisterNewUser(t *testing.T) {
	userService, repo, _, dispatcher := setup(t)

	t.Run("Success", func(t *testing.T) {
		email := "test@example.com"
		user, err := userService.RegisterNewUser("John", "Doe", email, "password123")

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, model.Active, user.Status)
		assert.Contains(t, user.HashedPassword, "-hashed")

		savedUser, _ := repo.FindByEmail(email)
		assert.Equal(t, user.ID, savedUser.ID)

		require.Len(t, dispatcher.events, 1)
		_, ok := dispatcher.events[0].(model.UserRegistered)
		assert.True(t, ok)
	})

	t.Run("Fail on email taken", func(t *testing.T) {
		dispatcher.Reset()
		_, err := userService.RegisterNewUser("Jane", "Doe", "test@example.com", "password123")
		assert.ErrorIs(t, err, model.ErrEmailTaken)
		assert.Empty(t, dispatcher.events)
	})

	t.Run("Fail on short password", func(t *testing.T) {
		dispatcher.Reset()
		_, err := userService.RegisterNewUser("Jack", "Smith", "jack@example.com", "123")
		assert.ErrorIs(t, err, service.ErrPasswordTooShort)
		assert.Empty(t, dispatcher.events)
	})
}

func TestSuspendUser(t *testing.T) {
	userService, repo, _, dispatcher := setup(t)
	user, _ := userService.RegisterNewUser("Initial", "User", "suspend@me.com", "longpassword")
	dispatcher.Reset()

	err := userService.SuspendUser(user.ID)

	require.NoError(t, err)
	updatedUser, _ := repo.Find(user.ID)
	assert.Equal(t, model.Suspended, updatedUser.Status)

	require.Len(t, dispatcher.events, 1)
	event, ok := dispatcher.events[0].(model.UserStatusChanged)
	require.True(t, ok)
	assert.Equal(t, model.Active, event.OldStatus)
	assert.Equal(t, model.Suspended, event.NewStatus)
}

type mockUserRepository struct {
	store map[uuid.UUID]*model.User
}

func (m *mockUserRepository) NextID() (uuid.UUID, error) { return uuid.New(), nil }
func (m *mockUserRepository) Create(user *model.User) error {
	m.store[user.ID] = user
	return nil
}
func (m *mockUserRepository) Update(user *model.User) error {
	m.store[user.ID] = user
	return nil
}
func (m *mockUserRepository) Find(id uuid.UUID) (*model.User, error) {
	if user, ok := m.store[id]; ok {
		return user, nil
	}
	return nil, model.ErrUserNotFound
}
func (m *mockUserRepository) FindByEmail(email string) (*model.User, error) {
	for _, user := range m.store {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, model.ErrUserNotFound
}

type mockPasswordManager struct{}

func (m *mockPasswordManager) Hash(pwd string) (string, error) {
	if pwd == "" {
		return "", errors.New("empty password")
	}
	return fmt.Sprintf("%s-hashed", pwd), nil
}
func (m *mockPasswordManager) Check(hashed, pwd string) (bool, error) {
	return hashed == fmt.Sprintf("%s-hashed", pwd), nil
}

type mockEventDispatcher struct {
	events []service.Event
}

func (m *mockEventDispatcher) Dispatch(event service.Event) error {
	m.events = append(m.events, event)
	return nil
}
func (m *mockEventDispatcher) Reset() {
	m.events = nil
}
