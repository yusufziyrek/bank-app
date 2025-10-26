package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              db,
		DialTimeout:     300 * time.Millisecond,
		ReadTimeout:     300 * time.Millisecond,
		WriteTimeout:    300 * time.Millisecond,
		PoolTimeout:     300 * time.Millisecond,
		MaxRetries:      1,
		MinRetryBackoff: 50 * time.Millisecond,
		MaxRetryBackoff: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}
