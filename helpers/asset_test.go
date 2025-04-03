package helpers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAssetDataCoupons(t *testing.T) {
	require := require.New(t)

	day0 := time.Now()
	day1 := time.Now().AddDate(0, 0, 1)
	day2 := time.Now().AddDate(0, 0, 2)
	day3 := time.Now().AddDate(0, 0, 3)
	day4 := time.Now().AddDate(0, 0, 4)
	day5 := time.Now().AddDate(0, 0, 5)

	ad := AssetData{}
	require.NoError(ad.RegisterAsset(
		"bondA",
		3,
		0.3, 0.4,
		false, true, true, true, true, false,
		[]*Coupon{
			// no coupons
		},
	))
	require.NoError(ad.RegisterAsset(
		"bondB",
		3,
		0.3, 0.4,
		false, true, true, true, true, false,
		[]*Coupon{
			// no coupons, has maturity one day from now
			{
				Date:       day1.Format(time.RFC1123),
				Start:      "",
				Yield:      1.0,
				IsMaturity: true,
			},
		},
	))
	require.NoError(ad.RegisterAsset(
		"bondC",
		3,
		0.3, 0.4,
		false, true, true, true, true, false,
		[]*Coupon{
			// two coupon periods, no maturity
			{
				Date:       day2.Format(time.RFC1123),
				Start:      day0.Format(time.RFC1123),
				Yield:      0.03,
				IsMaturity: false,
			},
			{
				Date:       day4.Format(time.RFC1123),
				Start:      day2.Format(time.RFC1123),
				Yield:      0.03,
				IsMaturity: false,
			},
		},
	))
	require.NoError(ad.RegisterAsset(
		"bondD",
		3,
		0.3, 0.4,
		false, true, true, true, true, false,
		[]*Coupon{
			// coupon period and maturity
			{
				Date:       day3.Format(time.RFC1123),
				Start:      day1.Format(time.RFC1123),
				Yield:      0.03,
				IsMaturity: false,
			},
			{
				Date:       day4.Format(time.RFC1123),
				Start:      "",
				Yield:      1.0,
				IsMaturity: true,
			},
		},
	))

	// CurrentCouponPeriod - Day 0 tests

	// No period found return zero values, no error
	requireNoCurrentCouponPeriod(t, ad, "bondA", day0)

	// Maturity tomorrow
	requireNoCurrentCouponPeriod(t, ad, "bondB", day0)

	// Coupon period starts today, so that is current period
	start, end, yield, err := ad.CurrentCouponPeriod("bondC", day0.Unix())
	require.NoError(err)
	require.Equal(0.03, yield)
	require.Equal(day0.Unix(), start)
	require.Equal(day2.Unix(), end)

	// Coupon period starts tomorrow
	requireNoCurrentCouponPeriod(t, ad, "bondD", day0)

	// CurrentCouponPeriod - Day 1 tests

	// No coupons
	requireNoCurrentCouponPeriod(t, ad, "bondA", day1)

	// Maturity today, but maturity is not considered a coupon period
	requireNoCurrentCouponPeriod(t, ad, "bondB", day1)

	// Middle of a coupon period
	start, end, yield, err = ad.CurrentCouponPeriod("bondC", day1.Unix())
	require.NoError(err)
	require.Equal(0.03, yield)
	require.Equal(day0.Unix(), start)
	require.Equal(day2.Unix(), end)

	// Coupon period starts today
	start, end, yield, err = ad.CurrentCouponPeriod("bondD", day1.Unix())
	require.NoError(err)
	require.Equal(0.03, yield)
	require.Equal(day1.Unix(), start)
	require.Equal(day3.Unix(), end)

	// CurrentCouponPeriod - Day 5 tests

	// Everything should have ended by day 5
	requireNoCurrentCouponPeriod(t, ad, "bondA", day5)
	requireNoCurrentCouponPeriod(t, ad, "bondB", day5)
	requireNoCurrentCouponPeriod(t, ad, "bondC", day5)
	requireNoCurrentCouponPeriod(t, ad, "bondD", day5)

	// CouponEndingAt tests

	requireNoCouponEndingAt(t, ad, "bondA", day0)
	requireNoCouponEndingAt(t, ad, "bondA", day1)
	requireNoCouponEndingAt(t, ad, "bondA", day2)
	requireNoCouponEndingAt(t, ad, "bondA", day3)
	requireNoCouponEndingAt(t, ad, "bondA", day4)
	requireNoCouponEndingAt(t, ad, "bondA", day5)

	requireNoCouponEndingAt(t, ad, "bondB", day0)
	requireCouponEndingAt(t, ad, "bondB", day1, 1.0) // maturity
	requireNoCouponEndingAt(t, ad, "bondB", day2)
	requireNoCouponEndingAt(t, ad, "bondB", day3)
	requireNoCouponEndingAt(t, ad, "bondB", day4)
	requireNoCouponEndingAt(t, ad, "bondB", day5)

	requireNoCouponEndingAt(t, ad, "bondC", day0)
	requireNoCouponEndingAt(t, ad, "bondC", day1)
	requireCouponEndingAt(t, ad, "bondC", day2, 0.03) // first coupon
	requireNoCouponEndingAt(t, ad, "bondC", day3)
	requireCouponEndingAt(t, ad, "bondC", day4, 0.03) // second coupon
	requireNoCouponEndingAt(t, ad, "bondC", day5)

	requireNoCouponEndingAt(t, ad, "bondD", day0)
	requireNoCouponEndingAt(t, ad, "bondD", day1)
	requireNoCouponEndingAt(t, ad, "bondD", day2)
	requireCouponEndingAt(t, ad, "bondD", day3, 0.03) // first coupon
	requireCouponEndingAt(t, ad, "bondD", day4, 1.0)  // maturity
	requireNoCouponEndingAt(t, ad, "bondD", day5)

	// CouponsEndingInRange tests

	// Large date range to capture all coupons and maturity payments
	requireCouponsEndingInRange(t, ad, "bondA", day0, day5, 0)
	requireCouponsEndingInRange(t, ad, "bondB", day0, day5, 1)
	requireCouponsEndingInRange(t, ad, "bondC", day0, day5, 2)
	requireCouponsEndingInRange(t, ad, "bondD", day0, day5, 2)

	// Reversed date range (return nothing)
	requireCouponsEndingInRange(t, ad, "bondA", day5, day0, 0)
	requireCouponsEndingInRange(t, ad, "bondB", day5, day0, 0)
	requireCouponsEndingInRange(t, ad, "bondC", day5, day0, 0)
	requireCouponsEndingInRange(t, ad, "bondD", day5, day0, 0)

	// Range ending on coupon date
	requireCouponsEndingInRange(t, ad, "bondB", day0, day1, 1)
	requireCouponsEndingInRange(t, ad, "bondC", day1, day2, 1)
	requireCouponsEndingInRange(t, ad, "bondD", day3, day4, 1)

	// Range starting on coupon date (return nothing)
	requireCouponsEndingInRange(t, ad, "bondB", day1, day2, 0)
	requireCouponsEndingInRange(t, ad, "bondC", day2, day3, 0)
	requireCouponsEndingInRange(t, ad, "bondD", day4, day5, 0)

	// Start time = end (return nothing, even if payment occurs on that date)
	requireCouponsEndingInRange(t, ad, "bondA", day0, day0, 0)
	requireCouponsEndingInRange(t, ad, "bondB", day1, day1, 0)
	requireCouponsEndingInRange(t, ad, "bondC", day2, day2, 0)
	requireCouponsEndingInRange(t, ad, "bondD", day4, day4, 0)
}

func requireNoCurrentCouponPeriod(t *testing.T, ad AssetData, bondID string, at time.Time) {
	t.Helper()
	start, end, yield, err := ad.CurrentCouponPeriod(bondID, at.Unix())
	require.NoError(t, err)
	require.Zero(t, yield)
	require.Zero(t, start)
	require.Zero(t, end)
}

func requireNoCouponEndingAt(t *testing.T, ad AssetData, bondID string, at time.Time) {
	t.Helper()
	c, err := ad.CouponEndingAt(bondID, at.Unix())
	require.Error(t, err)
	require.Nil(t, c)
}

func requireCouponEndingAt(t *testing.T, ad AssetData, bondID string, at time.Time, yield float64) {
	t.Helper()
	c, err := ad.CouponEndingAt(bondID, at.Unix())
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Equal(t, yield, c.Yield)
}

func requireCouponsEndingInRange(t *testing.T, ad AssetData, bondID string, from, to time.Time, quantity int) {
	t.Helper()
	cs, err := ad.CouponsEndingInRange(bondID, from.Unix(), to.Unix())
	require.NoError(t, err)
	require.NotNil(t, cs)
	require.Len(t, cs, quantity)
}
