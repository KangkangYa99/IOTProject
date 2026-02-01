package utils

import "context"

type Locker interface {
	Acquire(ctx context.Context, key string, ttlSeconds int) (bool, error)
	Release(ctx context.Context, key string) error
}
