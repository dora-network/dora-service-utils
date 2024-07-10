package redis

import (
	"context"
	redisv9 "github.com/redis/go-redis/v9"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Client
type Client interface {
	Close() error
	HGet(ctx context.Context, key, field string) *redisv9.StringCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redisv9.IntCmd
}
