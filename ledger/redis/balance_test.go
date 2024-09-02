package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dora-network/dora-service-utils/ledger/redis"
	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/ptr"
	"github.com/dora-network/dora-service-utils/testing/consts"
	"github.com/dora-network/dora-service-utils/testing/integration"
)

func TestBalances(t *testing.T) {
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
		"Should return empty balances if the user record doesn't exist", func(tt *testing.T) {
			balances, err := redis.GetUserBalances(ctx, rdb, time.Second, []string{consts.UserIDOne}, consts.StableID)
			require.NoError(tt, err)
			require.NotNil(tt, balances)
			emptyBalance := []types.Balance{
				{},
			}
			assert.Equal(tt, emptyBalance, balances)
		},
	)

	t.Run(
		"Should retrieve the users balances if it exists", func(tt *testing.T) {
			// first set up the user balances
			want := []types.Balance{
				{
					UserID:     consts.UserIDOne,
					AssetID:    consts.StableID,
					Balance:    100,
					Borrowed:   150,
					Collateral: 200,
					Supplied:   500,
					Virtual:    1000,
					Locked:     1350,
				},
			}

			require.NoError(
				t, rdb.HSet(
					ctx, redis.UserBalanceKey(consts.UserIDOne), consts.StableID,
					ptr.From(want[0]),
				).Err(),
			)
			balances, err := redis.GetUserBalances(ctx, rdb, time.Second, []string{consts.UserIDOne}, consts.StableID)
			require.NoError(tt, err)
			require.NotNil(tt, balances)
			assert.Equal(tt, want, balances)
		},
	)

	t.Run(
		"Should retrieve the balances for multiple users", func(tt *testing.T) {
			user2Balances := &types.Balance{
				UserID:     consts.UserIDTwo,
				AssetID:    consts.StableID,
				Balance:    1000,
				Borrowed:   1500,
				Collateral: 2000,
				Supplied:   5000,
				Virtual:    10000,
				Locked:     13500,
			}

			require.NoError(
				t,
				rdb.HSet(ctx, redis.UserBalanceKey(consts.UserIDTwo), consts.StableID, user2Balances).Err(),
			)
			balances, err := redis.GetUserBalances(
				ctx,
				rdb,
				time.Second,
				[]string{consts.UserIDOne, consts.UserIDTwo},
				consts.StableID,
			)
			require.NoError(tt, err)
			require.NotNil(tt, balances)
			want := []types.Balance{
				{
					UserID:     consts.UserIDOne,
					AssetID:    consts.StableID,
					Balance:    100,
					Borrowed:   150,
					Collateral: 200,
					Supplied:   500,
					Virtual:    1000,
					Locked:     1350,
				},
				{
					UserID:     consts.UserIDTwo,
					AssetID:    consts.StableID,
					Balance:    1000,
					Borrowed:   1500,
					Collateral: 2000,
					Supplied:   5000,
					Virtual:    10000,
					Locked:     13500,
				},
			}

			assert.Equal(tt, want, balances)
		},
	)

	t.Run(
		"Should update the user balances", func(tt *testing.T) {
			asset2Balances := &types.Balance{
				UserID:     consts.UserIDOne,
				AssetID:    consts.BondID,
				Balance:    90,
				Borrowed:   250,
				Collateral: 300,
				Supplied:   600,
				Virtual:    2000,
				Locked:     3300,
			}

			// first we want to set up the balances for asset 2
			require.NoError(
				tt,
				rdb.HSet(ctx, redis.UserBalanceKey(consts.UserIDOne), consts.BondID, asset2Balances).Err(),
			)

			balances, err := redis.GetUserBalances(
				ctx,
				rdb,
				time.Second,
				[]string{consts.UserIDOne},
				consts.StableID,
				consts.BondID,
			)
			require.NoError(tt, err)

			asset1 := balances[0]
			asset2 := balances[1]

			// update the balances
			asset1.Balance = 200
			asset2.Balance = 50

			require.NoError(
				t,
				redis.SetUserBalances(
					ctx,
					rdb,
					time.Second,
					map[string][]*types.Balance{consts.UserIDOne: {&asset1, &asset2}},
				),
			)

			// check if the balances were updated
			updated1 := new(types.Balance)
			updated2 := new(types.Balance)
			require.NoError(tt, rdb.HGet(ctx, redis.UserBalanceKey(consts.UserIDOne), consts.StableID).Scan(updated1))
			require.NoError(tt, rdb.HGet(ctx, redis.UserBalanceKey(consts.UserIDOne), consts.BondID).Scan(updated2))
			assert.Equal(tt, uint64(200), updated1.Balance)
			assert.Equal(tt, uint64(50), updated2.Balance)
			assert.Equal(tt, uint64(150), updated1.Borrowed)
			assert.Equal(tt, uint64(250), updated2.Borrowed)
		},
	)

	t.Run(
		"Should update module balances", func(tt *testing.T) {
			bondBalance := types.Balance{
				AssetID:    consts.BondID,
				Balance:    90,
				Borrowed:   250,
				Collateral: 300,
				Supplied:   600,
				Virtual:    2000,
				Locked:     3300,
			}

			stableBalance := types.Balance{
				AssetID:    consts.StableID,
				Balance:    990,
				Borrowed:   200,
				Collateral: 700,
				Supplied:   700,
				Virtual:    30,
				Locked:     66,
			}

			require.NoError(
				t,
				redis.SetModuleBalances(ctx, rdb, time.Second, []*types.Balance{&stableBalance, &bondBalance}),
			)

			balances, err := redis.GetModuleBalances(ctx, rdb, time.Second, consts.BondID, consts.StableID)
			require.NoError(tt, err)
			require.Len(tt, balances, 2)
			require.True(tt, balances[0].Equal(&bondBalance))
			require.True(tt, balances[1].Equal(&stableBalance))
		},
	)
}
