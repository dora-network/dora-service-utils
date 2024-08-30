package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dora-network/dora-service-utils/errors"
	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/testing/consts"
)

func TestUserLedger_Misc(t *testing.T) {
	t.Parallel()

	stableBalance := types.NewBalance(consts.UserIDOne, consts.StableID, 100, 120, 300, 500, 13, 1000)
	bondBalance := types.NewBalance(consts.UserIDOne, consts.BondID, 100, 120, 300, 500, 13, 1000)
	zeroBalance := types.ZeroBalance(consts.UserIDOne, "zeroAsset")
	l := types.NewUserLedger(consts.UserIDOne, stableBalance, bondBalance, zeroBalance)
	require.Equal(t, consts.UserIDOne, l.UserID())
	require.Len(t, l.AssetIDs(), 2)
	require.NoError(t, l.MustAssetIDs(consts.StableID, consts.BondID))
	require.Error(t, l.MustAssetIDs(consts.StableID, consts.BondID, "zero"))
	require.Error(t, l.MustAssetIDs("zero"))
	require.False(t, l.Has("zero"))
	require.True(t, l.Has(consts.StableID))
	slice := l.Slice()
	require.Len(t, slice, 2)
	// Alphabetical order on creation with sort
	require.Equal(t, bondBalance, slice[0])
	require.Equal(t, stableBalance, slice[1])
}

func TestUserLedger_Operations(t *testing.T) {
	t.Parallel()

	bondBalance := types.NewBalance(consts.UserIDOne, consts.BondID, 22, 12, 30, 30, 0, 0)
	l := types.NewUserLedger(consts.UserIDOne, bondBalance)

	// Add
	l, err := l.Add(
		types.ZeroAmount(consts.StableID),
		types.NewAmount(consts.StableID, 1000),
		types.NewAmount(consts.BondID, 11),
	)
	require.NoError(t, err)
	require.Len(t, l.AssetIDs(), 2)
	require.NoError(t, bondBalance.Add(types.NewAmount(consts.BondID, 11)))
	require.True(t, l.Select(consts.BondID).Equal(bondBalance))
	stableBalance := types.NewBalance(consts.UserIDOne, consts.StableID, 1000, 0, 0, 0, 0, 0)
	require.True(t, l.Select(consts.StableID).Equal(stableBalance))

	// Sub
	_, err = l.Sub(types.NewAmount(consts.BondID, 11), types.NewAmount(consts.StableID, 2000))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	_, err = l.Sub(types.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Sub(types.NewAmount(consts.BondID, 3), types.NewAmount(consts.StableID, 100))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			types.NewUserLedger(
				consts.UserIDOne,
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.StableID,
					Balance:    900,
					Borrowed:   0,
					Collateral: 0,
					Supplied:   0,
					Virtual:    0,
					Locked:     0,
				},
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.BondID,
					Balance:    30,
					Borrowed:   12,
					Collateral: 30,
					Supplied:   30,
					Virtual:    0,
					Locked:     0,
				},
			),
		),
	)

	// Lock
	_, err = l.Lock(types.NewAmount(consts.BondID, 11), types.NewAmount(consts.StableID, 1000))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	_, err = l.Lock(types.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Lock(types.NewAmount(consts.BondID, 5), types.NewAmount(consts.StableID, 200))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			types.NewUserLedger(
				consts.UserIDOne,
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.StableID,
					Balance:    700,
					Borrowed:   0,
					Collateral: 0,
					Supplied:   0,
					Virtual:    0,
					Locked:     200,
				},
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.BondID,
					Balance:    25,
					Borrowed:   12,
					Collateral: 30,
					Supplied:   30,
					Virtual:    0,
					Locked:     5,
				},
			),
		),
	)

	// Unlock
	_, err = l.Unlock(types.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	// no error on overflow on Unlock
	l, err = l.Unlock(types.NewAmount(consts.BondID, 1), types.NewAmount(consts.StableID, 201))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			types.NewUserLedger(
				consts.UserIDOne,
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.StableID,
					Balance:    900,
					Borrowed:   0,
					Collateral: 0,
					Supplied:   0,
					Virtual:    0,
					Locked:     0,
				},
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.BondID,
					Balance:    26,
					Borrowed:   12,
					Collateral: 30,
					Supplied:   30,
					Virtual:    0,
					Locked:     4,
				},
			),
		),
	)

	// Supply
	_, err = l.Supply(types.NewAmount(consts.BondID, 11), types.NewAmount(consts.StableID, 1000))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	_, err = l.Supply(types.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Supply(types.NewAmount(consts.BondID, 5), types.NewAmount(consts.StableID, 200))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			types.NewUserLedger(
				consts.UserIDOne,
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.StableID,
					Balance:    700,
					Borrowed:   0,
					Collateral: 0,
					Supplied:   200,
					Virtual:    0,
					Locked:     0,
				},
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.BondID,
					Balance:    21,
					Borrowed:   12,
					Collateral: 30,
					Supplied:   35,
					Virtual:    0,
					Locked:     4,
				},
			),
		),
	)

	// Withdraw
	_, err = l.Withdraw(types.NewAmount(consts.BondID, 40), types.NewAmount(consts.StableID, 10))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	_, err = l.Withdraw(types.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Withdraw(types.NewAmount(consts.BondID, 25), types.NewAmount(consts.StableID, 150))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			types.NewUserLedger(
				consts.UserIDOne,
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.StableID,
					Balance:    850,
					Borrowed:   0,
					Collateral: 0,
					Supplied:   50,
					Virtual:    0,
					Locked:     0,
				},
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.BondID,
					Balance:    46,
					Borrowed:   12,
					Collateral: 30,
					Supplied:   10,
					Virtual:    0,
					Locked:     4,
				},
			),
		),
	)

	// Borrow
	_, err = l.Borrow(types.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Borrow(types.NewAmount(consts.BondID, 25), types.NewAmount(consts.StableID, 150))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			types.NewUserLedger(
				consts.UserIDOne,
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.StableID,
					Balance:    1000,
					Borrowed:   150,
					Collateral: 0,
					Supplied:   50,
					Virtual:    0,
					Locked:     0,
				},
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.BondID,
					Balance:    71,
					Borrowed:   37,
					Collateral: 30,
					Supplied:   10,
					Virtual:    0,
					Locked:     4,
				},
			),
		),
	)

	// Repay
	_, err = l.Repay(types.NewAmount(consts.BondID, 40), types.NewAmount(consts.StableID, 160))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	_, err = l.Repay(types.NewAmount("not-present-asset", 11))
	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
	l, err = l.Repay(types.NewAmount(consts.BondID, 13), types.NewAmount(consts.StableID, 150))
	require.NoError(t, err)
	require.True(
		t, l.Equal(
			types.NewUserLedger(
				consts.UserIDOne,
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.StableID,
					Balance:    1000,
					Borrowed:   0,
					Collateral: 0,
					Supplied:   50,
					Virtual:    0,
					Locked:     0,
				},
				&types.Balance{
					UserID:     consts.UserIDOne,
					AssetID:    consts.BondID,
					Balance:    71,
					Borrowed:   24,
					Collateral: 30,
					Supplied:   10,
					Virtual:    0,
					Locked:     4,
				},
			),
		),
	)
}
