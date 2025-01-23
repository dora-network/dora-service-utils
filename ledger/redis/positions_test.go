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
			assert.Equal(tt, map[string]*types.Position{}, positions)
		},
	)

	t.Run(
		"Should set and retrieve user positions", func(tt *testing.T) {
			userPositions := map[string]*types.Position{
				consts.UserIDOne: {
					UserID:      consts.UserIDOne,
					Owned:       types.New(consts.StableID, 1000).Add(types.NewBalance(consts.BondID, 3)),
					Locked:      types.New(consts.StableID, 200),
					Supplied:    types.New(consts.BondID, 1),
					SSEQ:        types.Empty(),
					Inactive:    types.Empty(),
					NativeAsset: "USD",
					LastUpdated: time.Now().Unix(),
					Sequence:    1,
				},
				consts.UserIDTwo: {
					UserID:      consts.UserIDTwo,
					Owned:       types.New(consts.BondID, -122).Add(types.NewBalance(consts.StableID, 5000)),
					Locked:      types.New(consts.BondID, 100),
					Supplied:    types.New(consts.BondID, 22),
					SSEQ:        types.Empty(),
					Inactive:    types.Empty(),
					NativeAsset: "USD",
					LastUpdated: time.Now().Unix(),
					Sequence:    1,
				},
			}

			require.NoError(tt, redis.SetUsersPosition(ctx, rdb, time.Second, userPositions))

			positions, err := redis.GetUsersPosition(ctx, rdb, time.Second, consts.UserIDOne, consts.UserIDTwo)
			require.NoError(tt, err)
			require.NotNil(tt, positions)
			assert.Equal(
				tt, map[string]*types.Position{
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
			assert.Equal(tt, &types.Position{}, position)
		},
	)

	t.Run(
		"Should set and retrieve module position", func(tt *testing.T) {
			modulePosition := &types.Position{
				UserID:      "module",
				Owned:       types.New(consts.BondID, 1000).Add(types.NewBalance(consts.StableID, 50000)),
				Locked:      types.Empty(),
				Supplied:    types.New(consts.BondID, 1000).Add(types.NewBalance(consts.StableID, 50000)),
				SSEQ:        types.Empty(),
				Inactive:    types.Empty(),
				NativeAsset: "USD",
				LastUpdated: time.Now().Unix(),
				Sequence:    1,
			}

			require.NoError(tt, redis.SetModulePosition(ctx, rdb, time.Second, modulePosition))

			position, err := redis.GetModulePosition(ctx, rdb, time.Second)
			require.NoError(tt, err)
			assert.Equal(tt, modulePosition, position)
		},
	)
}
