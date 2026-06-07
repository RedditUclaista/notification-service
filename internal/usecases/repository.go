package usecases

import (
	"context"
	"github.com/RedditUclaista/notification-service/internal/entities"
)

type NotificationRepository interface {
	UpsertDeviceToken(ctx context.Context, device *entities.DeviceFCMToken) error
	GetDeviceToken(ctx context.Context, userID string) (*entities.DeviceFCMToken, error)
	DeleteDeviceToken(ctx context.Context, userID string) error

	InsertNotification(ctx context.Context, notification *entities.Notification) error
	GetNotificationsPaginated(ctx context.Context, userID string, limit int, offset int) ([]entities.Notification, int, error)
	MarkAsRead(ctx context.Context, id string, userID string) error
}
