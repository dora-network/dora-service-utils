package redis_test

import (
	"context"
	"github.com/dora-network/dora-service-utils/prices/redis"
	"github.com/dora-network/dora-service-utils/prices/types"
	"github.com/dora-network/dora-service-utils/ptr"
	"github.com/dora-network/dora-service-utils/testing/consts"
	"github.com/dora-network/dora-service-utils/testing/integration"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPrices(t *testing.T) {
	dn, err := integration.NewDoraNetwork(t)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, dn.Cleanup())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, dn.CreateRedisResource(t, ctx))

	rdb, err := dn.GetRedisClient()
	require.NoError(t, err)

	t.Run(
		"Should return empty price if asset's price record doesn't exist", func(tt *testing.T) {
			prices, err := redis.GetPrices(ctx, rdb, time.Second, consts.StableID)
			require.NoError(tt, err)
			require.NotNil(tt, prices)
			emptyPrices := []types.Price{
				{},
			}
			require.Equal(tt, emptyPrices, prices)
		},
	)

	t.Run(
		"Should retrieve the asset price if it exists", func(tt *testing.T) {
			stablePrice := types.Price{
				AssetID: consts.StableID,
				Price:   1.0,
			}

			require.NoError(
				t,
				rdb.HSet(
					ctx,
					redis.PricesKey(),
					consts.StableID,
					ptr.From(stablePrice),
				).Err(),
			)
			prices, err := redis.GetPrices(ctx, rdb, time.Second, consts.StableID)
			require.NoError(tt, err)
			require.NotNil(tt, prices)
			require.Equal(tt, []types.Price{stablePrice}, prices)
		},
	)

	t.Run(
		"Should retrieve prices for multiple assets", func(tt *testing.T) {
			bondPrice := types.Price{
				AssetID: consts.BondID,
				Price:   0.87,
			}

			require.NoError(
				t,
				rdb.HSet(
					ctx,
					redis.PricesKey(),
					consts.BondID,
					ptr.From(bondPrice),
				).Err(),
			)
			prices, err := redis.GetPrices(ctx, rdb, time.Second, consts.StableID, consts.BondID)
			require.NoError(tt, err)
			require.NotNil(tt, prices)
			want := []types.Price{
				{
					AssetID: consts.StableID,
					Price:   1.0,
				},
				{
					AssetID: consts.BondID,
					Price:   0.87,
				},
			}

			require.Equal(tt, want, prices)
		},
	)

	t.Run(
		"Should update bond price", func(tt *testing.T) {
			prices, err := redis.GetPrices(ctx, rdb, time.Second, consts.StableID, consts.BondID)
			require.NoError(tt, err)
			require.NotNil(tt, prices)
			require.Len(tt, prices, 2)

			match := 0
			for _, price := range prices {
				if price.AssetID == consts.BondID {
					require.Equal(tt, 0.87, price.Price)
					match++
				}
				if price.AssetID == consts.StableID {
					require.Equal(tt, 1.0, price.Price)
					match++
				}
			}
			require.Equal(tt, 2, match)

			require.NoError(
				t,
				redis.SetPrices(
					ctx,
					rdb,
					time.Second,
					map[string]float64{consts.BondID: 0.91},
				),
			)

			prices, err = redis.GetPrices(ctx, rdb, time.Second, consts.StableID, consts.BondID)
			require.NoError(tt, err)
			require.NotNil(tt, prices)
			require.Len(tt, prices, 2)
			match = 0
			for _, price := range prices {
				if price.AssetID == consts.BondID {
					require.Equal(tt, 0.91, price.Price)
					match++
				}
				if price.AssetID == consts.StableID {
					require.Equal(tt, 1.0, price.Price)
					match++
				}
			}
			require.Equal(tt, 2, match)
		},
	)
}
