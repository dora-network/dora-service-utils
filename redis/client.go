package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Client
type Client interface {
	redis.Cmdable
	Close() error
	Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error
}
