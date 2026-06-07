package database

import (
	"context"
	"fmt"
	"time"

	"github.com/RedditUclaista/notification-service/internal/entities"
)

func (r *DBRepo) InsertNotification(ctx context.Context, n *entities.Notification) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.%s (id, user_id, title, body, action_route, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, DB_SCHEMA, DB_TABLE_NOTIFICATION)

	createdAt := n.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := r.Cursor.ExecContext(ctx, query, n.ID, n.UserID, n.Title, n.Body, n.ActionRoute, n.IsRead, createdAt)
	if err != nil {
		return fmt.Errorf("failed to insert notification: %w", err)
	}
	return nil
}

func (r *DBRepo) GetNotificationsPaginated(ctx context.Context, userID string, limit int, offset int) ([]entities.Notification, int, error) {
	// 1. Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s.%s
		WHERE user_id = $1
	`, DB_SCHEMA, DB_TABLE_NOTIFICATION)

	var total int
	err := r.Cursor.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// 2. Get paginated notifications
	selectQuery := fmt.Sprintf(`
		SELECT id, user_id, title, body, action_route, is_read, created_at
		FROM %s.%s
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, DB_SCHEMA, DB_TABLE_NOTIFICATION)

	rows, err := r.Cursor.QueryContext(ctx, selectQuery, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	notifications := []entities.Notification{}
	for rows.Next() {
		var n entities.Notification
		err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.Title,
			&n.Body,
			&n.ActionRoute,
			&n.IsRead,
			&n.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification row: %w", err)
		}
		notifications = append(notifications, n)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error reading notification rows: %w", err)
	}

	return notifications, total, nil
}

func (r *DBRepo) MarkAsRead(ctx context.Context, id string, userID string) error {
	query := fmt.Sprintf(`
		UPDATE %s.%s
		SET is_read = true
		WHERE id = $1 AND user_id = $2
	`, DB_SCHEMA, DB_TABLE_NOTIFICATION)

	res, err := r.Cursor.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found or does not belong to this user")
	}

	return nil
}
