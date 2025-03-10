package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dora-network/dora-service-utils/ledger/redis"
	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/testing/consts"
	"github.com/dora-network/dora-service-utils/testing/integration"
)

func TestUserAndModulePosition_Redis(t *testing.T) {
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
		"Should return empty positions if the user record doesn't exist", func(tt *testing.T) {
			positions, err := redis.GetUsersPosition(ctx, rdb, time.Second, consts.UserIDOne, consts.UserIDTwo)
			require.NoError(tt, err)
			require.NotNil(tt, positions)
			assert.Equal(
				tt,
				map[string]*types.Position{
					consts.UserIDOne: types.InitialPosition(consts.UserIDOne),
					consts.UserIDTwo: types.InitialPosition(consts.UserIDTwo),
				},
				positions,
			)
		},
	)

	t.Run(
		"Should set and retrieve user positions", func(tt *testing.T) {
			userPositions := map[string]*types.Position{
				consts.UserIDOne: {
					UserID: consts.UserIDOne,
					Owned: types.NewBalances(consts.StableID, 1000).Add(
						types.NewBalance(
							consts.BondID,
							int64(3),
						),
					),
					Locked:      types.NewBalances(consts.StableID, 200),
					Supplied:    types.NewBalances(consts.BondID, 1),
					SSEQ:        types.EmptyBalances(),
					Inactive:    types.EmptyBalances(),
					NativeAsset: "USD",
					LastUpdated: time.Now().Unix(),
					Sequence:    1,
				},
				consts.UserIDTwo: {
					UserID: consts.UserIDTwo,
					Owned: types.NewBalances(consts.BondID, -122).Add(
						types.NewBalance(
							consts.StableID,
							int64(5000),
						),
					),
					Locked:      types.NewBalances(consts.BondID, 100),
					Supplied:    types.NewBalances(consts.BondID, 22),
					SSEQ:        types.EmptyBalances(),
					Inactive:    types.EmptyBalances(),
					NativeAsset: "USD",
					LastUpdated: time.Now().Unix(),
					Sequence:    1,
				},
			}

			require.NoError(tt, redis.SetUsersPosition(ctx, rdb, time.Second, userPositions))

			positions, err := redis.GetUsersPosition(ctx, rdb, time.Second, consts.UserIDOne, consts.UserIDTwo)
			require.NoError(tt, err)
			require.NotNil(tt, positions)
			assert.ObjectsAreEqual(
				map[string]*types.Position{
					consts.UserIDOne: userPositions[consts.UserIDOne],
					consts.UserIDTwo: userPositions[consts.UserIDTwo],
				}, positions,
			)
		},
	)

	t.Run(
		"Should return empty module position if it doesn't exist", func(tt *testing.T) {
			position, err := redis.GetModulePosition(ctx, rdb, time.Second)
			require.NoError(tt, err)
			assert.Equal(tt, types.InitialModule(), position)
		},
	)

	t.Run(
		"Should set and retrieve module position", func(tt *testing.T) {
			modulePosition := &types.Module{
				Balance: types.NewBalances(consts.BondID, 1000).Add(
					types.NewBalance(
						consts.StableID,
						int64(50000),
					),
				),
				Supplied: types.NewBalances(consts.BondID, 1000).Add(
					types.NewBalance(
						consts.StableID,
						int64(50000),
					),
				),
				Virtual:     types.EmptyBalances(),
				Borrowed:    types.EmptyBalances(),
				LastUpdated: time.Now().Unix(),
				Sequence:    1,
			}

			require.NoError(tt, redis.SetModulePosition(ctx, rdb, time.Second, modulePosition))

			position, err := redis.GetModulePosition(ctx, rdb, time.Second)
			require.NoError(tt, err)
			assert.ObjectsAreEqual(modulePosition, position)
		},
	)

	t.Run(
		"Should return all users positions keys", func(tt *testing.T) {
			keys, err := redis.GetAllUsersPositionKeys(ctx, rdb)
			require.NoError(tt, err)
			assert.ElementsMatch(tt, keys, []string{
				"positions:users:user1",
				"positions:users:user2",
			})
		},
	)
}
