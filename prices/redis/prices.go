package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/dora-network/dora-service-utils/prices/types"
	"time"

	"github.com/cenkalti/backoff/v4"
	redisv9 "github.com/redis/go-redis/v9"

	"github.com/dora-network/dora-service-utils/redis"
)

func PricesKey() string {
	return fmt.Sprint("prices")
}

func GetPrices(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	assetIDs ...string,
) ([]types.Price, error) {
	return getPrices(ctx, rdb, timeout, PricesKey(), assetIDs...)
}

func getPrices(ctx context.Context, rdb redis.Client, timeout time.Duration, keys string, ids ...string) (
	[]types.Price,
	error,
) {
	var prices []types.Price

	f := func(tx *redisv9.Tx) error {
		res, err := tx.HMGet(ctx, keys, ids...).Result()
		if err != nil {
			if errors.Is(err, redisv9.Nil) {
				return nil
			}
			return err
		}

		for _, v := range res {
			if v == nil {
				prices = append(prices, types.Price{})
				continue
			}

			b := new(types.Price)
			if err := b.UnmarshalBinary([]byte(v.(string))); err != nil {
				return err
			}
			prices = append(prices, *b)
		}

		return nil
	}

	if err := redis.TryTransaction(
		ctx,
		rdb,
		f,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		keys,
	); err != nil {
		return nil, err
	}

	return prices, nil
}

func GetPricesCmd(ctx context.Context, tx redis.Cmdable, assetIDs ...string) *redisv9.SliceCmd {
	watch := PricesKey()
	return tx.HMGet(ctx, watch, assetIDs...)
}

func SetPrices(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	reqs map[string]float64,
) error {
	watch := []string{PricesKey()}
	txFunc := func(tx *redisv9.Tx) error {
		values := make(map[string]any)
		for assetID, price := range reqs {
			values[assetID] = &types.Price{
				AssetID: assetID,
				Price:   price,
			}
		}
		// write the prices to redis
		err := tx.HSet(ctx, PricesKey(), values).Err()
		if err != nil {
			return err
		}

		return nil
	}

	return redis.TryTransaction(
		ctx,
		rdb,
		txFunc,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		watch...,
	)
}

func SetPricesCmd(ctx context.Context, tx redis.Cmdable, reqs map[string]float64) []redisv9.Cmder {
	cmds := make([]redisv9.Cmder, 0)
	values := make(map[string]any)
	for assetID, price := range reqs {
		values[assetID] = &types.Price{
			AssetID: assetID,
			Price:   price,
		}
	}

	// write the prices to redis
	cmd := tx.HSet(ctx, PricesKey(), values)
	cmds = append(cmds, cmd)

	return cmds
}
