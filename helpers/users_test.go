package helpers

import (
	"testing"
	"time"

	"github.com/dora-network/bond-api-golang/graph/types"
	ltypes "github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/stretchr/testify/require"
)

func TestUpdateCouponInterest(t *testing.T) {
	require := require.New(t)

	day0 := time.Now()
	day1 := day0.AddDate(0, 0, 1)
	day2 := day0.AddDate(0, 0, 2)
	day3 := day0.AddDate(0, 0, 3)
	day4 := day0.AddDate(0, 0, 4)
	day4plus := day4.Add(time.Hour)
	day5 := day0.AddDate(0, 0, 5)
	day6 := day0.AddDate(0, 0, 6)

	userA := "UserA" // will have balance of Asset A
	userB := "UserB" // with have borrow of Asset A
	assetA := "AssetA"

	assets := AssetData{}
	require.NoError(assets.RegisterAsset(
		assetA, 2, 0, 0,
		false, true, true, true, true, false,
		[]*Coupon{
			// coupon periods and maturity
			{
				Date:       day2.Format(time.RFC1123),
				Start:      day1.Format(time.RFC1123),
				Yield:      0.03,
				IsMaturity: false,
			},
			{
				Date:       day3.Format(time.RFC1123),
				Start:      day2.Format(time.RFC1123),
				Yield:      0.03,
				IsMaturity: false,
			},
			{
				Date:       day4.Format(time.RFC1123),
				Start:      day3.Format(time.RFC1123),
				Yield:      0.03,
				IsMaturity: false,
			},
			{
				Date:       day5.Format(time.RFC1123),
				Start:      "",
				Yield:      1.0,
				IsMaturity: true,
			},
		},
	))
	interest := interestAsset()
	require.NoError(assets.RegisterAsset(
		interest.UID,
		interest.Decimals,
		0.9,
		0.95,
		interest.Type == types.AssetTypeCurrency,
		interest.Type == types.AssetTypeBond,
		interest.HasUsage(types.AssetUsageTrade),
		interest.CanBorrow(),
		interest.HasUsage(types.AssetUsageVirtualBorrow),
		interest.IsInterest(),
		nil,
	))

	users := UserPositionTracker{}
	err := users.InitUserPosition(userA, ltypes.InitialPosition(userA))
	require.NoError(err)
	err = users.InitUserPosition(userB, ltypes.InitialPosition(userA))
	require.NoError(err)

	// Add a balance and a borrow on the users
	err = users.modifyUserBalances(userA, ltypes.NewBalance(assetA, int64(1000)), nil)
	require.NoError(err)
	err = users.modifyUserBalances(userB, nil, ltypes.NewBalance(assetA, int64(1000)))
	require.NoError(err)

	// No error, but nothing happens before coupon periods start
	earned, owed, err := users.UpdateCouponInterest(assets, userA, day0.Unix())
	requireCouponInterest(t, earned, owed, err, 0)

	// No error, but nothing happens before coupon periods start
	earned, owed, err = users.UpdateCouponInterest(assets, userB, day1.Unix())
	requireCouponInterest(t, earned, owed, err, 0)

	// First coupon period
	earned, owed, err = users.UpdateCouponInterest(assets, userA, day2.Unix())
	requireCouponInterest(t, earned, owed, err, 30)

	earned, owed, err = users.UpdateCouponInterest(assets, userB, day2.Unix())
	requireCouponInterest(t, earned, owed, err, -30)

	// Pass two more coupon periods at once
	earned, owed, err = users.UpdateCouponInterest(assets, userA, day4.Unix())
	requireCouponInterest(t, earned, owed, err, 60)

	earned, owed, err = users.UpdateCouponInterest(assets, userB, day4.Unix())
	requireCouponInterest(t, earned, owed, err, -60)

	// Elapse zero time, on a payment date
	earned, owed, err = users.UpdateCouponInterest(assets, userA, day4.Unix())
	requireCouponInterest(t, earned, owed, err, 0)

	earned, owed, err = users.UpdateCouponInterest(assets, userB, day4.Unix())
	requireCouponInterest(t, earned, owed, err, 0)

	// Elapse one hour (no payments)
	earned, owed, err = users.UpdateCouponInterest(assets, userA, day4plus.Unix())
	requireCouponInterest(t, earned, owed, err, 0)

	earned, owed, err = users.UpdateCouponInterest(assets, userB, day4plus.Unix())
	requireCouponInterest(t, earned, owed, err, 0)

	// Maturity date
	earned, owed, err = users.UpdateCouponInterest(assets, userA, day5.Unix())
	requireCouponInterest(t, earned, owed, err, 1000)

	earned, owed, err = users.UpdateCouponInterest(assets, userB, day5.Unix())
	requireCouponInterest(t, earned, owed, err, -1000)

	// After all payment dates (nothing happends)
	earned, owed, err = users.UpdateCouponInterest(assets, userA, day6.Unix())
	requireCouponInterest(t, earned, owed, err, 0)

	earned, owed, err = users.UpdateCouponInterest(assets, userB, day6.Unix())
	requireCouponInterest(t, earned, owed, err, 0)
}

func requireCouponInterest(t *testing.T, earned, owed *ltypes.Balance, err error, amt int64) {
	t.Helper()
	require := require.New(t)

	require.NoError(err)
	require.NotNil(earned)
	require.NotNil(owed)
	require.Equal("Interest", earned.Asset)
	require.Equal("Interest", owed.Asset)
	if amt > 0 {
		require.Equal(amt, int64(earned.Amount))
		require.Zero(owed.Amount)
	} else if amt < 0 {
		require.Zero(earned.Amount)
		require.Equal(-1*amt, int64(owed.Amount))
	} else {
		require.Zero(earned.Amount)
		require.Zero(owed.Amount)
	}
}

// TODO: de-duplicate (spanner pkg)

const (
	interestAssetID = "Interest"
)

// interestAsset is a hard-coded asset with ID "Interest" which represents a dollar amount of interest earned
func interestAsset() types.Asset {
	return types.Asset{
		UID:                  interestAssetID,
		Symbol:               interestAssetID,
		Decimals:             2,
		Description:          "Interest earned, represented in dollars with two decimal points",
		Type:                 types.AssetTypeInterest,
		Usage:                []types.AssetUsage{}, // not tradeable and cannot be borrowed
		CollateralWeight:     "0",
		LiquidationThreshold: "0",
		MaxUtilization:       "0",
		MaxSupply:            "0",
		Bond:                 nil,
		CreatedAt:            0,
	}
}
