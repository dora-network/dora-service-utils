package ledger_test

import (
	"context"
	"github.com/dora-network/dora-service-utils/errors"
	"github.com/dora-network/dora-service-utils/ledger"
	"github.com/dora-network/dora-service-utils/ptr"
	"github.com/dora-network/dora-service-utils/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestUserLedger_Misc(t *testing.T) {
	t.Parallel()

	stableBalance := ledger.NewBalance(UserIDOne, StableID, 100, 120, 300, 500, 13, 1000)
	bondBalance := ledger.NewBalance(UserIDOne, BondID, 100, 120, 300, 500, 13, 1000)
	zeroBalance := ledger.ZeroBalance(UserIDOne, "zeroAsset")
	l := ledger.NewUserLedger(UserIDOne, *stableBalance, *bondBalance, *zeroBalance)
	require.Equal(t, UserIDOne, l.UserID())
	require.Len(t, l.AssetIDs(), 2)
	require.NoError(t, l.MustAssetIDs(StableID, BondID))
	require.Error(t, l.MustAssetIDs(StableID, BondID, "zero"))
	require.Error(t, l.MustAssetIDs("zero"))
	require.False(t, l.Has("zero"))
	require.True(t, l.Has(StableID))
	slice := l.Slice()
	require.Len(t, slice, 2)
	// Alphabetical order on creation with sort
	require.Equal(t, bondBalance, slice[0])
	require.Equal(t, stableBalance, slice[1])
}

func TestUserLedger_Operations(t *testing.T) {
	t.Parallel()

	bondBalance := ledger.NewBalance(UserIDOne, BondID, 22, 12, 30, 30, 0, 0)
	l := ledger.NewUserLedger(UserIDOne, *bondBalance)

	// Add
	l, err := l.Add(ledger.ZeroAmount(StableID), ledger.NewAmount(StableID, 1000), ledger.NewAmount(BondID, 11))
	require.NoError(t, err)
	require.Len(t, l.AssetIDs(), 2)
	require.NoError(t, bondBalance.Add(ledger.NewAmount(BondID, 11)))
	require.True(t, l.Select(BondID).Equal(bondBalance))
	stableBalance := ledger.NewBalance(UserIDOne, StableID, 1000, 0, 0, 0, 0, 0)
	require.True(t, l.Select(StableID).Equal(stableBalance))

	// Sub
	_, err = l.Sub(ledger.NewAmount(BondID, 11), ledger.NewAmount(StableID, 2000))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	_, err = l.Sub(ledger.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Sub(ledger.NewAmount(BondID, 3), ledger.NewAmount(StableID, 100))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			ledger.NewUserLedger(
				UserIDOne,
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(StableID, 900),
					Borrowed:   ledger.ZeroAmount(StableID),
					Collateral: ledger.ZeroAmount(StableID),
					Supplied:   ledger.ZeroAmount(StableID),
					Virtual:    ledger.ZeroAmount(StableID),
					Locked:     ledger.ZeroAmount(StableID),
				},
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    BondID,
					Balance:    ledger.NewAmount(BondID, 30),
					Borrowed:   ledger.NewAmount(BondID, 12),
					Collateral: ledger.NewAmount(BondID, 30),
					Supplied:   ledger.NewAmount(BondID, 30),
					Virtual:    ledger.ZeroAmount(BondID),
					Locked:     ledger.ZeroAmount(BondID),
				},
			),
		),
	)

	// Lock
	_, err = l.Lock(ledger.NewAmount(BondID, 11), ledger.NewAmount(StableID, 1000))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	_, err = l.Lock(ledger.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Lock(ledger.NewAmount(BondID, 5), ledger.NewAmount(StableID, 200))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			ledger.NewUserLedger(
				UserIDOne,
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(StableID, 700),
					Borrowed:   ledger.ZeroAmount(StableID),
					Collateral: ledger.ZeroAmount(StableID),
					Supplied:   ledger.ZeroAmount(StableID),
					Virtual:    ledger.ZeroAmount(StableID),
					Locked:     ledger.NewAmount(StableID, 200),
				},
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    BondID,
					Balance:    ledger.NewAmount(BondID, 25),
					Borrowed:   ledger.NewAmount(BondID, 12),
					Collateral: ledger.NewAmount(BondID, 30),
					Supplied:   ledger.NewAmount(BondID, 30),
					Virtual:    ledger.ZeroAmount(BondID),
					Locked:     ledger.NewAmount(BondID, 5),
				},
			),
		),
	)

	// Unlock
	_, err = l.Unlock(ledger.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	// no error on overflow on Unlock
	l, err = l.Unlock(ledger.NewAmount(BondID, 1), ledger.NewAmount(StableID, 201))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			ledger.NewUserLedger(
				UserIDOne,
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(StableID, 900),
					Borrowed:   ledger.ZeroAmount(StableID),
					Collateral: ledger.ZeroAmount(StableID),
					Supplied:   ledger.ZeroAmount(StableID),
					Virtual:    ledger.ZeroAmount(StableID),
					Locked:     ledger.ZeroAmount(StableID),
				},
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    BondID,
					Balance:    ledger.NewAmount(BondID, 26),
					Borrowed:   ledger.NewAmount(BondID, 12),
					Collateral: ledger.NewAmount(BondID, 30),
					Supplied:   ledger.NewAmount(BondID, 30),
					Virtual:    ledger.ZeroAmount(BondID),
					Locked:     ledger.NewAmount(BondID, 4),
				},
			),
		),
	)

	// Supply
	_, err = l.Supply(ledger.NewAmount(BondID, 11), ledger.NewAmount(StableID, 1000))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	_, err = l.Supply(ledger.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Supply(ledger.NewAmount(BondID, 5), ledger.NewAmount(StableID, 200))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			ledger.NewUserLedger(
				UserIDOne,
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(StableID, 700),
					Borrowed:   ledger.ZeroAmount(StableID),
					Collateral: ledger.ZeroAmount(StableID),
					Supplied:   ledger.NewAmount(StableID, 200),
					Virtual:    ledger.ZeroAmount(StableID),
					Locked:     ledger.ZeroAmount(StableID),
				},
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    BondID,
					Balance:    ledger.NewAmount(BondID, 21),
					Borrowed:   ledger.NewAmount(BondID, 12),
					Collateral: ledger.NewAmount(BondID, 30),
					Supplied:   ledger.NewAmount(BondID, 35),
					Virtual:    ledger.ZeroAmount(BondID),
					Locked:     ledger.NewAmount(BondID, 4),
				},
			),
		),
	)

	// Withdraw
	_, err = l.Withdraw(ledger.NewAmount(BondID, 40), ledger.NewAmount(StableID, 10))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	_, err = l.Withdraw(ledger.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Withdraw(ledger.NewAmount(BondID, 25), ledger.NewAmount(StableID, 150))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			ledger.NewUserLedger(
				UserIDOne,
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(StableID, 850),
					Borrowed:   ledger.ZeroAmount(StableID),
					Collateral: ledger.ZeroAmount(StableID),
					Supplied:   ledger.NewAmount(StableID, 50),
					Virtual:    ledger.ZeroAmount(StableID),
					Locked:     ledger.ZeroAmount(StableID),
				},
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    BondID,
					Balance:    ledger.NewAmount(BondID, 46),
					Borrowed:   ledger.NewAmount(BondID, 12),
					Collateral: ledger.NewAmount(BondID, 30),
					Supplied:   ledger.NewAmount(BondID, 10),
					Virtual:    ledger.ZeroAmount(BondID),
					Locked:     ledger.NewAmount(BondID, 4),
				},
			),
		),
	)

	// Borrow
	_, err = l.Borrow(ledger.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Borrow(ledger.NewAmount(BondID, 25), ledger.NewAmount(StableID, 150))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			ledger.NewUserLedger(
				UserIDOne,
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(StableID, 1000),
					Borrowed:   ledger.NewAmount(StableID, 150),
					Collateral: ledger.ZeroAmount(StableID),
					Supplied:   ledger.NewAmount(StableID, 50),
					Virtual:    ledger.ZeroAmount(StableID),
					Locked:     ledger.ZeroAmount(StableID),
				},
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    BondID,
					Balance:    ledger.NewAmount(BondID, 71),
					Borrowed:   ledger.NewAmount(BondID, 37),
					Collateral: ledger.NewAmount(BondID, 30),
					Supplied:   ledger.NewAmount(BondID, 10),
					Virtual:    ledger.ZeroAmount(BondID),
					Locked:     ledger.NewAmount(BondID, 4),
				},
			),
		),
	)

	// Repay
	_, err = l.Repay(ledger.NewAmount(BondID, 40), ledger.NewAmount(StableID, 160))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	_, err = l.Repay(ledger.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Repay(ledger.NewAmount(BondID, 13), ledger.NewAmount(StableID, 150))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			ledger.NewUserLedger(
				UserIDOne,
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(StableID, 1000),
					Borrowed:   ledger.ZeroAmount(StableID),
					Collateral: ledger.ZeroAmount(StableID),
					Supplied:   ledger.NewAmount(StableID, 50),
					Virtual:    ledger.ZeroAmount(StableID),
					Locked:     ledger.ZeroAmount(StableID),
				},
				ledger.Balance{
					UserID:     UserIDOne,
					AssetID:    BondID,
					Balance:    ledger.NewAmount(BondID, 71),
					Borrowed:   ledger.NewAmount(BondID, 24),
					Collateral: ledger.NewAmount(BondID, 30),
					Supplied:   ledger.NewAmount(BondID, 10),
					Virtual:    ledger.ZeroAmount(BondID),
					Locked:     ledger.NewAmount(BondID, 4),
				},
			),
		),
	)
}

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
			l, err := ledger.GetUserLedger(ctx, rdb, time.Second, UserIDOne, UserIDTwo)
			require.NoError(tt, err)
			require.NotNil(tt, l)
			emptyLedger := []ledger.UserLedger{
				{},
			}
			assert.Equal(tt, emptyLedger, l)
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
		"Should retrieve the ledger for multiple users", func(tt *testing.T) {
			user1Balances := ledger.Balance{
				UserID:     UserIDOne,
				AssetID:    StableID,
				Balance:    ledger.NewAmount(StableID, 1000),
				Borrowed:   ledger.NewAmount(StableID, 1500),
				Collateral: ledger.NewAmount(StableID, 2000),
				Supplied:   ledger.NewAmount(StableID, 5000),
				Virtual:    ledger.NewAmount(StableID, 10000),
				Locked:     ledger.NewAmount(StableID, 13500),
			}

			user2Balances := []ledger.Balance{
				{
					UserID:     UserIDTwo,
					AssetID:    BondID,
					Balance:    ledger.NewAmount(BondID, 200),
					Borrowed:   ledger.NewAmount(BondID, 330),
					Collateral: ledger.NewAmount(BondID, 10),
					Supplied:   ledger.NewAmount(BondID, 400),
					Virtual:    ledger.NewAmount(BondID, 20),
					Locked:     ledger.NewAmount(BondID, 1350),
				},
				{
					UserID:     UserIDTwo,
					AssetID:    StableID,
					Balance:    ledger.NewAmount(BondID, 2000),
					Borrowed:   ledger.NewAmount(BondID, 500),
					Collateral: ledger.NewAmount(BondID, 220),
					Supplied:   ledger.NewAmount(BondID, 1000),
					Virtual:    ledger.NewAmount(BondID, 0),
					Locked:     ledger.NewAmount(BondID, 200),
				},
			}

			require.NoError(
				tt, ledger.SetUserBalances(
					ctx, rdb, time.Second, map[string][]ledger.Balance{
						UserIDOne: {user1Balances},
						UserIDTwo: user2Balances,
					},
				),
			)

			ledgers, err := ledger.GetUserLedger(ctx, rdb, time.Second, UserIDOne, UserIDTwo)
			require.NoError(tt, err)
			require.NotNil(tt, ledgers)
			require.Len(tt, ledgers, 2)
			want := []ledger.UserLedger{
				ledger.NewUserLedger(UserIDOne, user1Balances),
				ledger.NewUserLedger(UserIDTwo, user2Balances...),
			}

			assert.Equal(tt, want, ledgers)
		},
	)
}
