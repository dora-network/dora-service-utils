package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/dora-network/dora-service-utils/orderbook"
	"github.com/dora-network/dora-service-utils/pools/types"
	"github.com/dora-network/dora-service-utils/redis"
	redisv9 "github.com/redis/go-redis/v9"
	"time"
)

var poolKeys = []string{
	"pool_id",
	"base_asset",
	"quote_asset",
	"is_product_pool",
	"amount_shares",
	"amount_base",
	"amount_quote",
	"fee_factor",
	"created_at",
	"maturity_at",
}

func PoolKey(poolID string) string {
	return fmt.Sprintf("pools:%s", poolID)
}

func GetPools(ctx context.Context, rdb redis.Client, timeout time.Duration, poolIDs []string) ([]*types.Pool, error) {
	pools := make([]*types.Pool, 0)
	watch := make([]string, 0)

	f := func(tx *redisv9.Tx) error {
		cmd, err := tx.TxPipelined(
			ctx, func(pipe redisv9.Pipeliner) error {
				for _, poolID := range poolIDs {
					poolKey := PoolKey(poolID)
					watch = append(watch, poolKey)
					pipe.HGetAll(ctx, poolKey)
				}
				return nil
			},
		)
		if err != nil {
			return err
		}

		for _, c := range cmd {
			p := new(types.Pool)
			if err = c.(*redisv9.MapStringStringCmd).Scan(p); err != nil {
				return err
			}
			pools = append(pools, p)
		}

		return nil
	}

	if err := redis.TryTransaction(
		ctx,
		rdb,
		f,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		watch...,
	); err != nil {
		return nil, err
	}

	return pools, nil
}

func GetPool(ctx context.Context, rdb redis.Client, timeout time.Duration, poolID string) (*types.Pool, error) {
	watch := PoolKey(poolID)

	pool := new(types.Pool)

	f := func(tx *redisv9.Tx) error {
		err := tx.HMGet(ctx, watch, poolKeys...).Scan(pool)
		if err != nil {
			if errors.Is(err, redisv9.Nil) {
				return nil
			}
			return err
		}

		return nil
	}

	if err := redis.TryTransaction(
		ctx,
		rdb,
		f,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		watch,
	); err != nil {
		return nil, err
	}

	return pool, nil
}

func GetPoolsCmd(ctx context.Context, tx redis.Cmdable, poolIDs []string) ([]redisv9.Cmder, []string, error) {
	watch := make([]string, 0)
	cmds, err := tx.TxPipelined(
		ctx, func(pipe redisv9.Pipeliner) error {
			for _, poolID := range poolIDs {
				poolKey := PoolKey(poolID)
				watch = append(watch, poolKey)
				pipe.HGetAll(ctx, poolKey)
			}
			return nil
		},
	)
	if err != nil {
		return nil, nil, err
	}

	return cmds, watch, nil
}

func GetPoolCmd(ctx context.Context, tx redis.Cmdable, poolID string) (*redisv9.SliceCmd, string) {
	watch := PoolKey(poolID)
	return tx.HMGet(ctx, watch, poolKeys...), watch
}

func UpdatePool(ctx context.Context, rdb redis.Client, pool *types.Pool, timeout time.Duration, poolID string) error {
	txFunc := func(tx *redisv9.Tx) error {
		return tx.HSet(
			ctx, PoolKey(poolID),
			// We have to set each field individually rather than just passing the struct
			// which would be easier, because when serializing the struct, go-redis uses the
			// MarshalBinary method for the decimal.Decimal type (fee factor), but when
			// deserializing, it uses UnmarshalText which is expecting a number expressed
			// as a string. This causes the deserialization to fail
			"pool_id", pool.PoolID,
			"base_asset", pool.BaseAsset,
			"quote_asset", pool.QuoteAsset,
			"is_product_pool", pool.IsProductPool,
			"amount_shares", pool.AmountShares,
			"amount_base", pool.AmountBase,
			"amount_quote", pool.AmountQuote,
			"fee_factor", pool.FeeFactor.String(),
			"created_at", pool.CreatedAt,
			"maturity_at", pool.MaturityAt,
		).Err()
	}

	return redis.TryTransaction(
		ctx,
		rdb,
		txFunc,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		PoolKey(poolID),
	)
}

func UpdatePoolCmd(
	ctx context.Context,
	tx redis.Cmdable,
	pool *types.Pool,
) (*redisv9.IntCmd, string) {
	return tx.HSet(
		ctx, PoolKey(pool.PoolID),
		// We have to set each field individually rather than just passing the struct
		// which would be easier, because when serializing the struct, go-redis uses the
		// MarshalBinary method for the decimal.Decimal type (fee factor), but when
		// deserializing, it uses UnmarshalText which is expecting a number expressed
		// as a string. This causes the deserialization to fail
		"pool_id", pool.PoolID,
		"base_asset", pool.BaseAsset,
		"quote_asset", pool.QuoteAsset,
		"is_product_pool", pool.IsProductPool,
		"amount_shares", pool.AmountShares,
		"amount_base", pool.AmountBase,
		"amount_quote", pool.AmountQuote,
		"fee_factor", pool.FeeFactor.String(),
		"created_at", pool.CreatedAt,
		"maturity_at", pool.MaturityAt,
	), PoolKey(pool.PoolID)
}

func UpdatePoolBalance(
	ctx context.Context,
	rdb redis.Client,
	poolID string, amountShares, amountBase, amountQuote uint64,
	timeout time.Duration,
) error {
	txFunc := func(tx *redisv9.Tx) error {
		return tx.HSet(
			ctx, PoolKey(poolID),
			// We have to set each field individually rather than just passing the struct
			// which would be easier, because when serializing the struct, go-redis uses the
			// MarshalBinary method for the decimal.Decimal type (fee factor), but when
			// deserializing, it uses UnmarshalText which is expecting a number expressed
			// as a string. This causes the deserialization to fail
			"amount_shares", amountShares,
			"amount_base", amountBase,
			"amount_quote", amountQuote,
		).Err()
	}

	return redis.TryTransaction(
		ctx,
		rdb,
		txFunc,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		PoolKey(poolID),
	)
}

func UpdatePoolBalanceCmd(
	ctx context.Context, tx redis.Cmdable, poolID string, amountShares, amountBase, amountQuote uint64,
) (*redisv9.IntCmd, string) {
	return tx.HSet(
		ctx, PoolKey(poolID),
		// We have to set each field individually rather than just passing the struct
		// which would be easier, because when serializing the struct, go-redis uses the
		// MarshalBinary method for the decimal.Decimal type (fee factor), but when
		// deserializing, it uses UnmarshalText which is expecting a number expressed
		// as a string. This causes the deserialization to fail
		"amount_shares", amountShares,
		"amount_base", amountBase,
		"amount_quote", amountQuote,
	), PoolKey(poolID)
}

func CreatePool(ctx context.Context, rdb redis.Client, pool *types.Pool, timeout time.Duration) error {
	poolID := orderbook.ID(pool.BaseAsset, pool.QuoteAsset)
	pool.PoolID = poolID
	return UpdatePool(ctx, rdb, pool, timeout, poolID)
}
