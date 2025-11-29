package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"notification/pkg/domain/model"
)

type Event interface{ Type() string }
type EventDispatcher interface{ Dispatch(event Event) error }

type NotificationService interface {
	SendWelcomeEmail(userID uuid.UUID, email, firstName string) error
	NotifyOrderConfirmation(userID uuid.UUID, email string, orderID uuid.UUID) error
	NotifyPaymentFailed(userID uuid.UUID, email string, orderID uuid.UUID, reason string) error
}

func NewNotificationService(repo model.NotificationRepository, senders map[model.NotificationChannel]model.NotificationSender, dispatcher EventDispatcher) NotificationService {
	return &notificationService{repo: repo, senders: senders, dispatcher: dispatcher}
}

type notificationService struct {
	repo       model.NotificationRepository
	senders    map[model.NotificationChannel]model.NotificationSender
	dispatcher EventDispatcher
}

func (s *notificationService) SendWelcomeEmail(userID uuid.UUID, email, firstName string) error {
	subject := "Welcome to our store!"
	body := fmt.Sprintf("Hi %s, thanks for joining us!", firstName)

	return s.orchestrateSend(userID, email, subject, body, model.Email)
}

func (s *notificationService) NotifyOrderConfirmation(userID uuid.UUID, email string, orderID uuid.UUID) error {
	subject := fmt.Sprintf("Your order %s has been confirmed!", orderID.String())
	body := "We have received your order and will process it shortly."

	return s.orchestrateSend(userID, email, subject, body, model.Email)
}

func (s *notificationService) NotifyPaymentFailed(userID uuid.UUID, email string, orderID uuid.UUID, reason string) error {
	subject := fmt.Sprintf("Payment failed for order %s", orderID.String())
	body := fmt.Sprintf("Unfortunately, the payment for your order failed. Reason: %s", reason)

	return s.orchestrateSend(userID, email, subject, body, model.Email)
}

func (s *notificationService) orchestrateSend(userID uuid.UUID, recipient, subject, body string, channel model.NotificationChannel) error {
	notifID, err := s.repo.NextID()
	if err != nil {
		return err
	}
	notification := &model.Notification{
		ID:               notifID,
		UserID:           userID,
		Channel:          channel,
		RecipientAddress: recipient,
		Subject:          subject,
		Body:             body,
		Status:           model.Pending,
		CreatedAt:        time.Now().UTC(),
	}
	if err := s.repo.Create(notification); err != nil {
		return err
	}

	sender, ok := s.senders[channel]
	if !ok {
		return fmt.Errorf("no sender configured for channel %v", channel)
	}

	err = sender.Send(recipient, subject, body)

	if err != nil {
		notification.Status = model.Failed
		notification.FailureReason = err.Error()
		_ = s.dispatcher.Dispatch(model.NotificationFailed{
			NotificationID: notifID, UserID: userID, Channel: channel, Reason: err.Error(),
		})
	} else {
		now := time.Now().UTC()
		notification.Status = model.Sent
		notification.SentAt = &now
		_ = s.dispatcher.Dispatch(model.NotificationSent{
			NotificationID: notifID, UserID: userID, Channel: channel,
		})
	}

	return s.repo.Update(notification)
}
