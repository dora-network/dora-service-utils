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

func TestUserLedger_Redis(t *testing.T) {
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
		"Should return empty ledger if the user record doesn't exist", func(tt *testing.T) {
			l, err := redis.GetUserLedger(ctx, rdb, time.Second, consts.UserIDOne, consts.UserIDTwo)
			require.NoError(tt, err)
			require.NotNil(tt, l)
			emptyLedger := []types.UserLedger{
				{},
				{},
			}
			assert.Equal(tt, emptyLedger, l)
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
				t,
				rdb.HSet(ctx, redis.UserBalanceKey(consts.UserIDOne), consts.StableID, ptr.From(want[0])).Err(),
			)
			balances, err := redis.GetUserBalances(ctx, rdb, time.Second, []string{consts.UserIDOne}, consts.StableID)
			require.NoError(tt, err)
			require.NotNil(tt, balances)
			assert.Equal(tt, want, balances)
		},
	)

	t.Run(
		"Should retrieve the ledger for multiple users", func(tt *testing.T) {
			user1Balances := &types.Balance{
				UserID:     consts.UserIDOne,
				AssetID:    consts.StableID,
				Balance:    1000,
				Borrowed:   1500,
				Collateral: 2000,
				Supplied:   5000,
				Virtual:    10000,
				Locked:     13500,
			}

			user2Balances := []*types.Balance{
				{
					UserID:     consts.UserIDTwo,
					AssetID:    consts.BondID,
					Balance:    200,
					Borrowed:   330,
					Collateral: 10,
					Supplied:   400,
					Virtual:    20,
					Locked:     1350,
				},
				{
					UserID:     consts.UserIDTwo,
					AssetID:    consts.StableID,
					Balance:    2000,
					Borrowed:   500,
					Collateral: 220,
					Supplied:   1000,
					Virtual:    0,
					Locked:     200,
				},
			}

			require.NoError(
				tt, redis.SetUserBalances(
					ctx, rdb, time.Second, map[string][]*types.Balance{
						consts.UserIDOne: {user1Balances},
						consts.UserIDTwo: user2Balances,
					},
				),
			)

			ledgers, err := redis.GetUserLedger(ctx, rdb, time.Second, consts.UserIDOne, consts.UserIDTwo)
			require.NoError(tt, err)
			require.NotNil(tt, ledgers)
			require.Len(tt, ledgers, 2)
			want := []types.UserLedger{
				types.NewUserLedger(consts.UserIDOne, user1Balances),
				types.NewUserLedger(consts.UserIDTwo, user2Balances...),
			}

			assert.Equal(tt, want, ledgers)
		},
	)
}
