package pools_test

import (
	"context"
	"testing"
	"time"

	"github.com/dora-network/dora-service-utils/orderbook"
	"github.com/stretchr/testify/assert"

	"github.com/dora-network/dora-service-utils/pools"
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

	want := pools.Pool{
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

	t.Run("Should save a pool to Redis", func(tt *testing.T) {
		require.NoError(t, pools.CreatePool(ctx, rdb, &want, time.Second))

		fields := []string{
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
		var got pools.Pool
		require.NoError(t, rdb.HMGet(ctx, pools.PoolBalanceKey(orderbook.ID(want.BaseAsset, want.QuoteAsset)), fields...).Scan(&got))
		assert.Equal(tt, want, got)
	})

	t.Run("Should retrieve a pool from Redis", func(tt *testing.T) {
		got, err := pools.GetPoolBalances(ctx, rdb, time.Second, orderbook.ID(want.BaseAsset, want.QuoteAsset))
		require.NoError(tt, err)
		assert.Equal(tt, want, *got)
	})

	t.Run("Should update a pool in Redis", func(tt *testing.T) {
		updated := pools.Pool{
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

		require.NoError(t, pools.UpdatePool(ctx, rdb, &updated, time.Second, pools.PoolBalanceKey(orderbook.ID(updated.BaseAsset, updated.QuoteAsset))))
		got, err := pools.GetPoolBalances(ctx, rdb, time.Second, orderbook.ID(updated.BaseAsset, updated.QuoteAsset))
		require.NoError(tt, err)
		assert.Equal(tt, updated, *got)
	})
}
