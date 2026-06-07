package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	glide "github.com/valkey-io/valkey-glide/go/v2"
	"github.com/valkey-io/valkey-glide/go/v2/config"
)

type NotificationCache struct {
	client *glide.Client
}

func NewNotificationCache(host string, port int, dbIndex int, timeout time.Duration) (*NotificationCache, error) {
	clientConfig := config.NewClientConfiguration().
		WithAddress(&config.NodeAddress{Host: host, Port: port}).
		WithRequestTimeout(timeout).
		WithDatabaseId(dbIndex)

	client, err := glide.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache client: %w", err)
	}

	return &NotificationCache{client: client}, nil
}

func (c *NotificationCache) IncrementVotes(ctx context.Context, userID, postID string) (int64, error) {
	key := fmt.Sprintf("buffer:%s:%s:votes", userID, postID)
	val, err := c.client.Incr(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("failed to increment votes in cache: %w", err)
	}
	return val, nil
}

func (c *NotificationCache) IncrementComments(ctx context.Context, userID, postID string) (int64, error) {
	key := fmt.Sprintf("buffer:%s:%s:comments", userID, postID)
	val, err := c.client.Incr(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("failed to increment comments in cache: %w", err)
	}
	return val, nil
}

func (c *NotificationCache) AcquireLock(ctx context.Context, userID, postID string, ttl time.Duration) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s:%s", userID, postID)

	// Consultamos si ya existe el bloqueo en caché
	result, err := c.client.Get(ctx, lockKey)
	if err != nil {
		return false, fmt.Errorf("failed to check lock status: %w", err)
	}

	// Si no es nulo, significa que el temporizador de 2 segundos ya está corriendo
	if !result.IsNil() {
		return false, nil
	}

	// De lo contrario, adquirimos el bloqueo asignándole valor "1"
	_, err = c.client.Set(ctx, lockKey, "1")
	if err != nil {
		return false, fmt.Errorf("failed to set lock: %w", err)
	}

	// Asignamos el tiempo de vida (TTL) del bloqueo
	_, err = c.client.Expire(ctx, lockKey, ttl)
	if err != nil {
		return false, fmt.Errorf("failed to set expiry for lock: %w", err)
	}

	return true, nil
}

func (c *NotificationCache) GetCountsAndClear(ctx context.Context, userID, postID string) (int, int, error) {
	votesKey := fmt.Sprintf("buffer:%s:%s:votes", userID, postID)
	commentsKey := fmt.Sprintf("buffer:%s:%s:comments", userID, postID)

	// Obtenemos los votos
	votesRes, err := c.client.Get(ctx, votesKey)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get votes count: %w", err)
	}

	// Obtenemos los comentarios
	commentsRes, err := c.client.Get(ctx, commentsKey)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get comments count: %w", err)
	}

	votes := 0
	if !votesRes.IsNil() {
		votes, _ = strconv.Atoi(votesRes.Value())
	}

	comments := 0
	if !commentsRes.IsNil() {
		comments, _ = strconv.Atoi(commentsRes.Value())
	}

	// Borramos las claves de acumulación para la próxima ventana
	_, err = c.client.Del(ctx, []string{votesKey, commentsKey})
	if err != nil {
		fmt.Printf("warning: failed to delete buffer keys: %v\n", err)
	}

	return votes, comments, nil
}
