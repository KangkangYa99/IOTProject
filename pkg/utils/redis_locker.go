package utils

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLocker struct {
	redis *redis.Client
}

func NewRedisLocker(redis *redis.Client) *RedisLocker {
	return &RedisLocker{
		redis: redis,
	}
}
func (r *RedisLocker) Acquire(ctx context.Context, key string, ttlSeconds int) (bool, error) {
	ok, err := r.redis.SetNX(ctx, key, "1", time.Duration(ttlSeconds)*time.Second).Result()
	return ok, err
}
func (r *RedisLocker) Release(ctx context.Context, key string) error {
	return r.redis.Del(ctx, key).Err()
}
