package lib

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type PushServiceClient struct {
	client   *messaging.Client
	mockMode bool
}

func NewPushServiceClient(credentialsPath string, mockMode bool) (*PushServiceClient, error) {
	if mockMode {
		fmt.Println("FCM: Initializing push service in MOCK mode (simulated prints).")
		return &PushServiceClient{mockMode: true}, nil
	}

	// Verificamos si el archivo de credenciales de Firebase existe
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		fmt.Printf("FCM: Warnings - service-account file not found at %s. Falling back to MOCK mode.\n", credentialsPath)
		return &PushServiceClient{mockMode: true}, nil
	}

	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase app: %w", err)
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize fcm messaging client: %w", err)
	}

	fmt.Println("FCM: Official Firebase Admin SDK initialized successfully.")
	return &PushServiceClient{client: client, mockMode: false}, nil
}

// SendPushNotification envía una notificación push.
// Retorna (true, nil) si fue exitoso.
// Retorna (false, nil) si el token es inválido/expirado (debe ser borrado de la base de datos).
// Retorna (false, err) si hubo un error del servidor o de conexión.
func (s *PushServiceClient) SendPushNotification(ctx context.Context, token, title, body, actionRoute string) (bool, error) {
	if s.mockMode {
		fmt.Printf("\n--- [FCM PUSH NOTIFICATION (SIMULATION)] ---\nToken: %s\nTitle: %s\nBody: %s\nAction Route: %s\n--------------------------------------------\n\n", token, title, body, actionRoute)
		return true, nil
	}

	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: map[string]string{
			"action_route": actionRoute,
		},
	}

	_, err := s.client.Send(ctx, message)
	if err != nil {
		// Verificamos si el error indica que el token es inválido o ya no está registrado
		if messaging.IsRegistrationTokenNotRegistered(err) {
			fmt.Printf("FCM: Token has expired or is unregistered. Invalidating from DB.\n")
			return false, nil
		}
		return false, fmt.Errorf("failed to send push notification via FCM: %w", err)
	}

	return true, nil
}
