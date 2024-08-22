package balance_test

import (
	"context"
	"testing"
	"time"

	"github.com/dora-network/dora-service-utils/ptr"

	redisv9 "github.com/redis/go-redis/v9"

	"github.com/dora-network/dora-service-utils/balance"
	"github.com/stretchr/testify/assert"

	"github.com/dora-network/dora-service-utils/testing/integration"
	"github.com/stretchr/testify/require"
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

	t.Run("Should return empty balances if the user record doesn't exist", func(tt *testing.T) {
		balances, err := balance.GetUserBalances(ctx, rdb, time.Second, []string{"user1"}, "asset1")
		require.NoError(tt, err)
		require.NotNil(tt, balances)
		emptyBalance := []balance.Balances{
			{},
		}
		assert.Equal(tt, emptyBalance, balances)
	})

	t.Run("Should retrieve the users balances if it exists", func(tt *testing.T) {
		// first set up the user balances
		want := []balance.Balances{
			{
				UserID:  "user1",
				AssetID: "asset1",
				Balance: balance.Balance{
					Amount:    100,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Borrowed: balance.Balance{
					Amount:    150,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Collateral: balance.Balance{
					Amount:    200,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Supplied: balance.Balance{
					Amount:    500,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Virtual: balance.Balance{
					Amount:    1000,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
			},
		}

		require.NoError(t, rdb.HSet(ctx, balance.UserBalanceKey("user1"), "asset1", ptr.From(want[0])).Err())
		balances, err := balance.GetUserBalances(ctx, rdb, time.Second, []string{"user1"}, "asset1")
		require.NoError(tt, err)
		require.NotNil(tt, balances)
		assert.Equal(tt, want, balances)
	})

	t.Run("Should retrieve the balances for multiple users", func(tt *testing.T) {
		user2Balances := &balance.Balances{
			UserID:  "user2",
			AssetID: "asset1",
			Balance: balance.Balance{
				Amount:    1000,
				Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
			},
			Borrowed: balance.Balance{
				Amount:    1500,
				Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
			},
			Collateral: balance.Balance{
				Amount:    2000,
				Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
			},
			Supplied: balance.Balance{
				Amount:    5000,
				Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
			},
			Virtual: balance.Balance{
				Amount:    10000,
				Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
			},
		}

		require.NoError(t, rdb.HSet(ctx, balance.UserBalanceKey("user2"), "asset1", user2Balances).Err())
		balances, err := balance.GetUserBalances(ctx, rdb, time.Second, []string{"user1", "user2"}, "asset1")
		require.NoError(tt, err)
		require.NotNil(tt, balances)
		want := []balance.Balances{
			{
				UserID:  "user1",
				AssetID: "asset1",
				Balance: balance.Balance{
					Amount:    100,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Borrowed: balance.Balance{
					Amount:    150,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Collateral: balance.Balance{
					Amount:    200,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Supplied: balance.Balance{
					Amount:    500,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Virtual: balance.Balance{
					Amount:    1000,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
			},
			{
				UserID:  "user2",
				AssetID: "asset1",
				Balance: balance.Balance{
					Amount:    1000,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Borrowed: balance.Balance{
					Amount:    1500,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Collateral: balance.Balance{
					Amount:    2000,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Supplied: balance.Balance{
					Amount:    5000,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
				Virtual: balance.Balance{
					Amount:    10000,
					Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
				},
			},
		}

		assert.Equal(tt, want, balances)
	})

	t.Run("Should update the user balances", func(tt *testing.T) {
		asset2Balances := &balance.Balances{
			UserID:  "user1",
			AssetID: "asset2",
			Balance: balance.Balance{
				Amount:    90,
				Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
			},
			Borrowed: balance.Balance{
				Amount:    250,
				Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
			},
			Collateral: balance.Balance{
				Amount:    300,
				Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
			},
			Supplied: balance.Balance{
				Amount:    600,
				Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
			},
			Virtual: balance.Balance{
				Amount:    2000,
				Timestamp: time.Date(2024, 8, 12, 20, 0, 0, 0, time.UTC),
			},
		}

		// first we want to set up the balances for asset 2
		require.NoError(tt, rdb.HSet(ctx, balance.UserBalanceKey("user1"), "asset2", asset2Balances).Err())

		txFunc := func(tx *redisv9.Tx) error {
			// the operation should retrieve the balances for asset 1 and asset 2,
			// update the balances for both and then write them back to redis
			res, err := tx.HMGet(ctx, balance.UserBalanceKey("user1"), "asset1", "asset2").Result()
			if err != nil {
				return err
			}

			balances := make([]*balance.Balances, 0, len(res))
			for _, v := range res {
				if v == nil {
					balances = append(balances, &balance.Balances{})
					continue
				}
				b := new(balance.Balances)
				if err := b.UnmarshalBinary([]byte(v.(string))); err != nil {
					return err
				}
				balances = append(balances, b)
			}

			asset1 := balances[0]
			asset2 := balances[1]

			// update the balances
			asset1.Balance.Amount = 200
			asset2.Balance.Amount = 50

			// write the balances back to redis
			err = tx.HSet(ctx, balance.UserBalanceKey("user1"), "asset1", asset1, "asset2", asset2).Err()
			if err != nil {
				return err
			}

			return nil
		}

		require.NoError(t, balance.UpdateBalances(ctx, rdb, txFunc, time.Second, balance.UserBalanceKey("user1")))
		// check if the balances were updated
		updated1 := new(balance.Balances)
		updated2 := new(balance.Balances)
		require.NoError(tt, rdb.HGet(ctx, balance.UserBalanceKey("user1"), "asset1").Scan(updated1))
		require.NoError(tt, rdb.HGet(ctx, balance.UserBalanceKey("user1"), "asset2").Scan(updated2))
		assert.Equal(tt, uint64(200), updated1.Balance.Amount)
		assert.Equal(tt, uint64(50), updated2.Balance.Amount)
		assert.Equal(tt, uint64(150), updated1.Borrowed.Amount)
		assert.Equal(tt, uint64(250), updated2.Borrowed.Amount)
	})
}
