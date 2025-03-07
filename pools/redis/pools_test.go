package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/dora-network/dora-service-utils/pools/redis"

	"github.com/dora-network/dora-service-utils/orderbook"
	"github.com/stretchr/testify/assert"

	"github.com/dora-network/dora-service-utils/pools/types"
	"github.com/govalues/decimal"

	"github.com/dora-network/dora-service-utils/testing/integration"
	"github.com/stretchr/testify/require"
)

var timeout = 10 * time.Second

func TestPools(t *testing.T) {
	dn, err := integration.NewDoraNetwork(t)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, dn.Cleanup())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	require.NoError(t, dn.CreateRedisResource(t, ctx))

	rdb, err := dn.GetRedisClient()
	require.NoError(t, err)

	want := types.Pool{
		PoolID:        "base-quote",
		BaseAsset:     "base",
		QuoteAsset:    "quote",
		IsProductPool: true,
		AmountShares:  1000000,
		AmountBase:    1000000,
		AmountQuote:   1000000,
		FeeFactor:     decimal.MustNew(1, 2),
		CreatedAt:     time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC).UnixMilli(),
		MaturityAt:    time.Date(2034, 8, 12, 20, 0, 0, 0, time.UTC).UnixMilli(),
	}

	want1 := types.Pool{
		PoolID:        "base2-quote2",
		BaseAsset:     "base2",
		QuoteAsset:    "quote2",
		IsProductPool: true,
		AmountShares:  1000000,
		AmountBase:    1000000,
		AmountQuote:   1000000,
		FeeFactor:     decimal.MustNew(1, 2),
		CreatedAt:     time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC).UnixMilli(),
		MaturityAt:    time.Date(2034, 8, 12, 20, 0, 0, 0, time.UTC).UnixMilli(),
	}

	t.Run(
		"Should save a pool to Redis", func(tt *testing.T) {
			require.NoError(t, redis.CreatePool(ctx, rdb, &want, time.Second))

			fields := []string{
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
				"fees_collected_base",
				"fees_collected_quote",
			}
			var got types.Pool
			require.NoError(
				t,
				rdb.HMGet(
					ctx,
					redis.PoolKey(orderbook.ID(want.BaseAsset, want.QuoteAsset)),
					fields...,
				).Scan(&got),
			)
			assert.Equal(tt, want, got)
		},
	)

	t.Run(
		"Should retrieve a pool from Redis", func(tt *testing.T) {
			got, err := redis.GetPool(ctx, rdb, time.Second, orderbook.ID(want.BaseAsset, want.QuoteAsset))
			require.NoError(tt, err)
			assert.Equal(tt, want, *got)
		},
	)

	t.Run(
		"Should save 2 pools to Redis and list them", func(tt *testing.T) {
			require.NoError(t, redis.CreatePool(ctx, rdb, &want, time.Second))
			require.NoError(t, redis.CreatePool(ctx, rdb, &want1, time.Second))

			got, err := redis.GetPools(
				ctx, rdb, time.Minute, []string{
					orderbook.ID(want.BaseAsset, want.QuoteAsset),
					orderbook.ID(want1.BaseAsset, want1.QuoteAsset),
				},
			)
			require.NoError(tt, err)
			assert.Len(tt, got, 2)
			assert.Equal(tt, want, *got[0])
			assert.Equal(tt, want1, *got[1])
		},
	)

	t.Run(
		"Should update a pool in Redis", func(tt *testing.T) {
			updated := types.Pool{
				PoolID:             "base-quote",
				BaseAsset:          "base",
				QuoteAsset:         "quote",
				IsProductPool:      true,
				AmountShares:       1000000,
				AmountBase:         1000000,
				AmountQuote:        1000000,
				FeeFactor:          decimal.MustNew(1, 2),
				CreatedAt:          time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC).UnixMilli(),
				MaturityAt:         time.Date(2034, 8, 12, 20, 0, 0, 0, time.UTC).UnixMilli(),
				FeesCollectedBase:  0,
				FeesCollectedQuote: 0,
			}

			require.NoError(
				t,
				redis.UpdatePool(
					ctx,
					rdb,
					&updated,
					time.Second,
					orderbook.ID(updated.BaseAsset, updated.QuoteAsset),
				),
			)
			got, err := redis.GetPool(
				ctx,
				rdb,
				time.Second,
				orderbook.ID(updated.BaseAsset, updated.QuoteAsset),
			)
			require.NoError(tt, err)
			assert.Equal(tt, updated, *got)
		},
	)

	t.Run(
		"Should update a pool balance in Redis", func(tt *testing.T) {
			initial := types.Pool{
				PoolID:             "base-quote",
				BaseAsset:          "base",
				QuoteAsset:         "quote",
				IsProductPool:      true,
				AmountShares:       1000000,
				AmountBase:         1000000,
				AmountQuote:        1000000,
				FeeFactor:          decimal.MustNew(1, 2),
				CreatedAt:          time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC).UnixMilli(),
				MaturityAt:         time.Date(2034, 8, 12, 20, 0, 0, 0, time.UTC).UnixMilli(),
				FeesCollectedBase:  0,
				FeesCollectedQuote: 0,
			}

			require.NoError(
				t,
				redis.UpdatePool(
					ctx,
					rdb,
					&initial,
					time.Second,
					initial.PoolID,
				),
			)

			updated := types.Pool{
				AmountShares:       10000002,
				AmountBase:         10000001,
				AmountQuote:        10000001,
				FeesCollectedBase:  1000,
				FeesCollectedQuote: 1000,
			}

			require.NoError(
				t,
				redis.UpdatePoolBalance(ctx, rdb, "base-quote", 10000002, 10000001, 10000001, 0, 0, time.Second),
			)
			got, err := redis.GetPool(
				ctx,
				rdb,
				time.Second,
				"base-quote",
			)
			require.NoError(tt, err)
			assert.Equal(tt, updated.AmountQuote, got.AmountQuote)
			assert.Equal(tt, updated.AmountBase, got.AmountBase)
			assert.Equal(tt, updated.AmountShares, got.AmountShares)
			assert.Equal(tt, updated.FeesCollectedBase, updated.FeesCollectedBase)
			assert.Equal(tt, updated.FeesCollectedQuote, updated.FeesCollectedQuote)
			assert.Equal(tt, initial.PoolID, got.PoolID)
			assert.Equal(tt, initial.BaseAsset, got.BaseAsset)
			assert.Equal(tt, initial.QuoteAsset, got.QuoteAsset)
			assert.Equal(tt, initial.IsProductPool, got.IsProductPool)
			assert.Equal(tt, initial.FeeFactor, got.FeeFactor)
			assert.Equal(tt, initial.CreatedAt, got.CreatedAt)
			assert.Equal(tt, initial.MaturityAt, got.MaturityAt)
		},
	)
}
