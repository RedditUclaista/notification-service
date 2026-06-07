package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/RedditUclaista/notification-service/internal/entities"
)

func (r *DBRepo) UpsertDeviceToken(ctx context.Context, device *entities.DeviceFCMToken) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.%s (user_id, fcm_token, platform, last_updated)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id)
		DO UPDATE SET fcm_token = EXCLUDED.fcm_token, platform = EXCLUDED.platform, last_updated = EXCLUDED.last_updated
	`, DB_SCHEMA, DB_TABLE_DEVICE)

	_, err := r.Cursor.ExecContext(ctx, query, device.UserID, device.FCMToken, device.Platform, time.Now())
	if err != nil {
		return fmt.Errorf("failed to upsert device token: %w", err)
	}
	return nil
}

func (r *DBRepo) GetDeviceToken(ctx context.Context, userID string) (*entities.DeviceFCMToken, error) {
	query := fmt.Sprintf(`
		SELECT user_id, fcm_token, platform, last_updated
		FROM %s.%s
		WHERE user_id = $1
	`, DB_SCHEMA, DB_TABLE_DEVICE)

	device := &entities.DeviceFCMToken{}
	err := r.Cursor.QueryRowContext(ctx, query, userID).Scan(
		&device.UserID,
		&device.FCMToken,
		&device.Platform,
		&device.LastUpdated,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No token registered
		}
		return nil, fmt.Errorf("failed to get device token: %w", err)
	}
	return device, nil
}

func (r *DBRepo) DeleteDeviceToken(ctx context.Context, userID string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.%s
		WHERE user_id = $1
	`, DB_SCHEMA, DB_TABLE_DEVICE)

	_, err := r.Cursor.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete device token: %w", err)
	}
	return nil
}
