package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/RedditUclaista/notification-service/internal/dto"
	"github.com/RedditUclaista/notification-service/internal/entities"
	"github.com/google/uuid"
)

func (uc *notificationUseCase) RegisterDevice(ctx context.Context, req *dto.DeviceRegisterRequest) error {
	device := &entities.DeviceFCMToken{
		UserID:      req.UserID,
		FCMToken:    req.FCMToken,
		Platform:    req.Platform,
		LastUpdated: time.Now(),
	}

	return uc.repo.UpsertDeviceToken(ctx, device)
}

func (uc *notificationUseCase) GetNotifications(ctx context.Context, userID string, limit int, offset int) (*dto.NotificationHistoryResponse, error) {
	notifications, total, err := uc.repo.GetNotificationsPaginated(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user notifications: %w", err)
	}

	return &dto.NotificationHistoryResponse{
		Data: notifications,
		Meta: dto.Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	}, nil
}

func (uc *notificationUseCase) MarkAsRead(ctx context.Context, notificationID string, userID string) error {
	return uc.repo.MarkAsRead(ctx, notificationID, userID)
}

// BufferEvent incrementa los contadores en Valkey y programa el vaciado (flush) tras 2 segundos si es el primer evento.
func (uc *notificationUseCase) BufferEvent(ctx context.Context, userID string, postID string, eventType string) error {
	var err error
	if eventType == "vote" {
		_, err = uc.cache.IncrementVotes(ctx, userID, postID)
	} else if eventType == "comment" {
		_, err = uc.cache.IncrementComments(ctx, userID, postID)
	} else {
		return fmt.Errorf("unknown event type: %s", eventType)
	}

	if err != nil {
		return fmt.Errorf("failed to buffer event in cache: %w", err)
	}

	// Intentamos adquirir un bloqueo de 2 segundos.
	// Si da true, significa que es la primera alerta para este post y no hay temporizadores activos.
	acquired, err := uc.cache.AcquireLock(ctx, userID, postID, 2*time.Second)
	if err != nil {
		return fmt.Errorf("failed to acquire aggregation lock: %w", err)
	}

	if acquired {
		// Lanzamos una Goroutine que esperará los 2 segundos para vaciar el acumulador
		go func(uID, pID string) {
			time.Sleep(2 * time.Second)
			uc.flushNotification(context.Background(), uID, pID)
		}(userID, postID)
	}

	return nil
}

// flushNotification consolida las alertas acumuladas, las persiste en Postgres y envía la alerta push.
func (uc *notificationUseCase) flushNotification(ctx context.Context, userID, postID string) {
	votes, comments, err := uc.cache.GetCountsAndClear(ctx, userID, postID)
	if err != nil {
		fmt.Printf("Error clearing buffer counts: %v\n", err)
		return
	}

	if votes == 0 && comments == 0 {
		return // No hay actividad que reportar
	}

	var title, body string
	if votes > 0 && comments > 0 {
		title = "Actividad en tu publicación"
		body = fmt.Sprintf("Tu post ha recibido %d voto(s) y %d comentario(s).", votes, comments)
	} else if votes > 0 {
		if votes == 1 {
			title = "¡Nuevo voto en tu publicación!"
			body = "A un estudiante le gustó tu post."
		} else {
			title = "¡Nuevos votos en tu publicación!"
			body = fmt.Sprintf("Tu post ha recibido %d votos.", votes)
		}
	} else if comments > 0 {
		if comments == 1 {
			title = "¡Nuevo comentario en tu publicación!"
			body = "Un estudiante ha comentado en tu post."
		} else {
			title = "¡Nuevos comentarios en tu publicación!"
			body = fmt.Sprintf("%d personas han comentado en tu post.", comments)
		}
	}

	actionRoute := fmt.Sprintf("/posts/v1/details/%s", postID)

	notification := &entities.Notification{
		ID:          uuid.New().String(),
		UserID:      userID,
		Title:       title,
		Body:        body,
		ActionRoute: actionRoute,
		IsRead:      false,
		CreatedAt:   time.Now(),
	}

	// 1. Guardar la alerta consolidada en la Base de Datos relacional para el historial
	err = uc.repo.InsertNotification(ctx, notification)
	if err != nil {
		fmt.Printf("Error inserting consolidated notification: %v\n", err)
		return
	}

	// 2. Buscar si el usuario destino tiene un token registrado para enviar Push
	device, err := uc.repo.GetDeviceToken(ctx, userID)
	if err != nil {
		fmt.Printf("Error fetching device token: %v\n", err)
		return
	}

	if device != nil && device.FCMToken != "" {
		// 3. Disparar notificación push mediante FCM SDK
		success, err := uc.pushClient.SendPushNotification(ctx, device.FCMToken, title, body, actionRoute)
		if err != nil {
			fmt.Printf("Error sending push notification: %v\n", err)
			return
		}

		// Si Firebase reporta que el token expiró o ya no es válido, lo borramos inmediatamente
		if !success {
			fmt.Printf("FCM: Invalid token detected for user %s. Deleting token.\n", userID)
			err = uc.repo.DeleteDeviceToken(ctx, userID)
			if err != nil {
				fmt.Printf("Error deleting expired device token: %v\n", err)
			}
		}
	}
}
