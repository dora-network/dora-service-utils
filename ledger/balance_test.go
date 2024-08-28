package ledger_test

import (
	"context"
	"github.com/dora-network/dora-service-utils/ledger"
	"testing"
	"time"

	"github.com/dora-network/dora-service-utils/ptr"

	redisv9 "github.com/redis/go-redis/v9"

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

	t.Run(
		"Should return empty balances if the user record doesn't exist", func(tt *testing.T) {
			balances, err := ledger.GetUserBalances(ctx, rdb, time.Second, []string{UserIDOne}, StableID)
			require.NoError(tt, err)
			require.NotNil(tt, balances)
			emptyBalance := []ledger.Balance{
				{},
			}
			assert.Equal(tt, emptyBalance, balances)
		},
	)

	t.Run(
		"Should retrieve the users balances if it exists", func(tt *testing.T) {
			// first set up the user balances
			want := []ledger.Balance{
				{
					UserID:     UserIDOne,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(StableID, 100),
					Borrowed:   ledger.NewAmount(StableID, 150),
					Collateral: ledger.NewAmount(StableID, 200),
					Supplied:   ledger.NewAmount(StableID, 500),
					Virtual:    ledger.NewAmount(StableID, 1000),
					Locked:     ledger.NewAmount(StableID, 1350),
				},
			}

			require.NoError(t, rdb.HSet(ctx, ledger.UserBalanceKey(UserIDOne), StableID, ptr.From(want[0])).Err())
			balances, err := ledger.GetUserBalances(ctx, rdb, time.Second, []string{UserIDOne}, StableID)
			require.NoError(tt, err)
			require.NotNil(tt, balances)
			assert.Equal(tt, want, balances)
		},
	)

	t.Run(
		"Should retrieve the balances for multiple users", func(tt *testing.T) {
			user2Balances := &ledger.Balance{
				UserID:     UserIDTwo,
				AssetID:    StableID,
				Balance:    ledger.NewAmount(StableID, 1000),
				Borrowed:   ledger.NewAmount(StableID, 1500),
				Collateral: ledger.NewAmount(StableID, 2000),
				Supplied:   ledger.NewAmount(StableID, 5000),
				Virtual:    ledger.NewAmount(StableID, 10000),
				Locked:     ledger.NewAmount(StableID, 13500),
			}

			require.NoError(t, rdb.HSet(ctx, ledger.UserBalanceKey(UserIDTwo), StableID, user2Balances).Err())
			balances, err := ledger.GetUserBalances(ctx, rdb, time.Second, []string{UserIDOne, UserIDTwo}, StableID)
			require.NoError(tt, err)
			require.NotNil(tt, balances)
			want := []ledger.Balance{
				{
					UserID:     UserIDOne,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(StableID, 100),
					Borrowed:   ledger.NewAmount(StableID, 150),
					Collateral: ledger.NewAmount(StableID, 200),
					Supplied:   ledger.NewAmount(StableID, 500),
					Virtual:    ledger.NewAmount(StableID, 1000),
					Locked:     ledger.NewAmount(StableID, 1350),
				},
				{
					UserID:     UserIDTwo,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(StableID, 1000),
					Borrowed:   ledger.NewAmount(StableID, 1500),
					Collateral: ledger.NewAmount(StableID, 2000),
					Supplied:   ledger.NewAmount(StableID, 5000),
					Virtual:    ledger.NewAmount(StableID, 10000),
					Locked:     ledger.NewAmount(StableID, 13500),
				},
			}

			assert.Equal(tt, want, balances)
		},
	)

	t.Run(
		"Should update the user balances", func(tt *testing.T) {
			asset2Balances := &ledger.Balance{
				UserID:     UserIDOne,
				AssetID:    BondID,
				Balance:    ledger.NewAmount(BondID, 90),
				Borrowed:   ledger.NewAmount(BondID, 250),
				Collateral: ledger.NewAmount(BondID, 300),
				Supplied:   ledger.NewAmount(BondID, 600),
				Virtual:    ledger.NewAmount(BondID, 2000),
				Locked:     ledger.NewAmount(BondID, 3300),
			}

			// first we want to set up the balances for asset 2
			require.NoError(tt, rdb.HSet(ctx, ledger.UserBalanceKey(UserIDOne), BondID, asset2Balances).Err())

			txFunc := func(tx *redisv9.Tx) error {
				// the operation should retrieve the balances for asset 1 and asset 2,
				// update the balances for both and then write them back to redis
				res, err := tx.HMGet(ctx, ledger.UserBalanceKey(UserIDOne), StableID, BondID).Result()
				if err != nil {
					return err
				}

				balances := make([]*ledger.Balance, 0, len(res))
				for _, v := range res {
					if v == nil {
						balances = append(balances, &ledger.Balance{})
						continue
					}
					b := new(ledger.Balance)
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
				err = tx.HSet(ctx, ledger.UserBalanceKey(UserIDOne), StableID, asset1, BondID, asset2).Err()
				if err != nil {
					return err
				}

				return nil
			}

			require.NoError(t, ledger.UpdateBalances(ctx, rdb, txFunc, time.Second, ledger.UserBalanceKey(UserIDOne)))
			// check if the balances were updated
			updated1 := new(ledger.Balance)
			updated2 := new(ledger.Balance)
			require.NoError(tt, rdb.HGet(ctx, ledger.UserBalanceKey(UserIDOne), StableID).Scan(updated1))
			require.NoError(tt, rdb.HGet(ctx, ledger.UserBalanceKey(UserIDOne), BondID).Scan(updated2))
			assert.Equal(tt, uint64(200), updated1.Balance.Amount)
			assert.Equal(tt, uint64(50), updated2.Balance.Amount)
			assert.Equal(tt, uint64(150), updated1.Borrowed.Amount)
			assert.Equal(tt, uint64(250), updated2.Borrowed.Amount)
		},
	)
}
