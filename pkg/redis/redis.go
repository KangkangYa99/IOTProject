package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis(addr string, password string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}
	RedisClient = client
	return client, nil
}
