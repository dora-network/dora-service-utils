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

// Cmdable wraps the `redis.Cmdable` interface, we include this to allow us to mock the `redis.Cmdable` interface
// but also so if we need to replace the redis client library, downstream consumers of this package don't need to
//
//counterfeiter:generate . Cmdable
type Cmdable interface {
	redis.Cmdable
}
