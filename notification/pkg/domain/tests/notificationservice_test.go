package tests

import (
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"notification/pkg/domain/model"
	"notification/pkg/domain/service"
	"testing"
)

func setup(t *testing.T) (service.NotificationService, *mockNotificationRepository, *mockNotificationSender, *mockEventDispatcher) {
	repo := &mockNotificationRepository{store: make(map[uuid.UUID]*model.Notification)}
	sender := &mockNotificationSender{}
	dispatcher := &mockEventDispatcher{}

	senders := map[model.NotificationChannel]model.NotificationSender{
		model.Email: sender,
	}

	notificationService := service.NewNotificationService(repo, senders, dispatcher)
	return notificationService, repo, sender, dispatcher
}

func TestSendWelcomeEmail(t *testing.T) {
	notificationService, repo, sender, dispatcher := setup(t)

	t.Run("Success path", func(t *testing.T) {
		sender.ShouldError = false
		dispatcher.Reset()

		userID := uuid.New()
		email := "test@example.com"
		err := notificationService.SendWelcomeEmail(userID, email, "John")

		require.NoError(t, err)

		assert.Equal(t, 1, sender.SendCount)
		assert.Equal(t, email, sender.LastRecipient)

		require.Len(t, repo.store, 1)
		var savedNotif *model.Notification
		for _, n := range repo.store {
			savedNotif = n
		}
		assert.Equal(t, model.Sent, savedNotif.Status)
		assert.NotNil(t, savedNotif.SentAt)

		require.Len(t, dispatcher.events, 1)
		_, ok := dispatcher.events[0].(model.NotificationSent)
		assert.True(t, ok)
	})

	t.Run("Sender fails", func(t *testing.T) {
		sender.ShouldError = true
		dispatcher.Reset()

		err := notificationService.SendWelcomeEmail(uuid.New(), "fail@example.com", "Jane")
		require.NoError(t, err)

		var savedNotif *model.Notification
		for _, n := range repo.store {
			if n.RecipientAddress == "fail@example.com" {
				savedNotif = n
			}
		}
		require.NotNil(t, savedNotif)
		assert.Equal(t, model.Failed, savedNotif.Status)
		assert.Equal(t, "failed to send", savedNotif.FailureReason)

		require.Len(t, dispatcher.events, 1)
		_, ok := dispatcher.events[0].(model.NotificationFailed)
		assert.True(t, ok)
	})
}

type mockNotificationRepository struct {
	store map[uuid.UUID]*model.Notification
}

func (m *mockNotificationRepository) NextID() (uuid.UUID, error) { return uuid.New(), nil }
func (m *mockNotificationRepository) Create(n *model.Notification) error {
	m.store[n.ID] = n
	return nil
}
func (m *mockNotificationRepository) Update(n *model.Notification) error {
	m.store[n.ID] = n
	return nil
}

type mockNotificationSender struct {
	SendCount     int
	LastRecipient string
	ShouldError   bool
}

func (m *mockNotificationSender) Send(recipient, subject, body string) error {
	m.SendCount++
	m.LastRecipient = recipient
	if m.ShouldError {
		return errors.New("failed to send")
	}
	return nil
}

type mockEventDispatcher struct {
	events []service.Event
}

func (m *mockEventDispatcher) Dispatch(event service.Event) error {
	m.events = append(m.events, event)
	return nil
}
func (m *mockEventDispatcher) Reset() { m.events = nil }
