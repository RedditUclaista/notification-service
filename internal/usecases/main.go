package usecases

import (
	"context"

	"github.com/RedditUclaista/notification-service/internal/cache"
	"github.com/RedditUclaista/notification-service/internal/dto"
	"github.com/RedditUclaista/notification-service/internal/lib"
)

type NotificationUseCase interface {
	RegisterDevice(ctx context.Context, req *dto.DeviceRegisterRequest) error
	GetNotifications(ctx context.Context, userID string, limit int, offset int) (*dto.NotificationHistoryResponse, error)
	MarkAsRead(ctx context.Context, notificationID string, userID string) error

	// Método para registrar eventos de LavinMQ y activar el buffer
	BufferEvent(ctx context.Context, userID string, postID string, eventType string) error
}

type notificationUseCase struct {
	repo       NotificationRepository
	cache      *cache.NotificationCache
	pushClient *lib.PushServiceClient
}

func NewNotificationUseCase(repo NotificationRepository, cache *cache.NotificationCache, pushClient *lib.PushServiceClient) NotificationUseCase {
	return &notificationUseCase{
		repo:       repo,
		cache:      cache,
		pushClient: pushClient,
	}
}
