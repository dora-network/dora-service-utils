package pools

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dora-network/dora-service-utils/orderbook"

	"github.com/cenkalti/backoff/v4"
	"github.com/dora-network/dora-service-utils/redis"
	"github.com/govalues/decimal"
	redisv9 "github.com/redis/go-redis/v9"
)

var poolKeys = []string{
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

// Pool represents a liquidity pool in the DORA network.
// This struct is for serialization purposes only.
type Pool struct {
	BaseAsset     string          `json:"base_asset" redis:"base_asset"`
	QuoteAsset    string          `json:"quote_asset" redis:"quote_asset"`
	IsProductPool bool            `json:"is_product_pool" redis:"is_product_pool"`
	AmountShares  uint64          `json:"amount_shares" redis:"amount_shares"`
	AmountBase    uint64          `json:"amount_base" redis:"amount_base"`
	AmountQuote   uint64          `json:"amount_quote" redis:"amount_quote"`
	FeeFactor     decimal.Decimal `json:"fee_factor" redis:"fee_factor"`
	CreatedAt     int64           `json:"created_at" redis:"created_at"`
	MaturityAt    int64           `json:"maturity_at" redis:"maturity_at"`
}

func PoolBalanceKey(poolID string) string {
	return fmt.Sprintf("pools:%s", poolID)
}

func GetPoolBalances(ctx context.Context, rdb redis.Client, timeout time.Duration, poolID string) (*Pool, error) {
	watch := PoolBalanceKey(poolID)

	pool := new(Pool)

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

func UpdatePool(ctx context.Context, rdb redis.Client, pool *Pool, timeout time.Duration, watch ...string) error {
	poolID := orderbook.ID(pool.BaseAsset, pool.QuoteAsset)

	txFunc := func(tx *redisv9.Tx) error {
		return tx.HSet(ctx, PoolBalanceKey(poolID),
			// We have to set each field individually rather than just passing the struct
			// which would be easier, because when serializing the struct, go-redis uses the
			// MarshalBinary method for the decimal.Decimal type (fee factor), but when
			// deserializing, it uses UnmarshalText which is expecting a number expressed
			// as a string. This causes the deserialization to fail
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
		watch...,
	)
}

func CreatePool(ctx context.Context, rdb redis.Client, pool *Pool, timeout time.Duration) error {
	poolID := orderbook.ID(pool.BaseAsset, pool.QuoteAsset)
	return UpdatePool(ctx, rdb, pool, timeout, poolID)
}
