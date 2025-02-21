package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssetID(t *testing.T) {
	require := require.New(t)

	validAssetIDs := []string{
		"Bond_A",
		"USD",
		"Bond_A-USD",          // pool share
		"Bond_A-Coupon_123",   // not an asset, but a valid Position.InterestSources entry
		"Bond_A-Snapshot_123", // not an asset, but a valid Module.DollarCouponFundSources entry
	}

	invalidAssetIDs := []string{
		"",
		"A&B Co.",             // only alphanumeric characters and underscores allowed, plus up to one hyphen
		"-abc",                // no leading hyphens
		"abc-",                // no trailing hyphens
		"Bond_A-USD-Bond_B",   // a single hyphen indicates pool share. multiple are forbidden
		"Coupon_123-Bond_B",   // coupon period must be after the hyphen for Position.InterestSources entry
		"Snapshot_123-Bond_B", // same rule; supply snapshots
		"A-A",                 // bond paired with itself
	}

	for _, id := range validAssetIDs {
		require.Nil(ValidAssetID(id), id)
	}

	for _, id := range invalidAssetIDs {
		require.Error(ValidAssetID(id), id)
	}

	bals := EmptyBalances().Add(
		NewBalance("Bond_A-Snapshot_1", int64(1)),
		NewBalance("Bond_B-Snapshot_2", int64(2)),
		NewBalance("Bond_C-Snapshot_3", int64(3)),
		NewBalance("Bond_A-Snapshot_4", int64(4)),
	).Sub(
		NewBalance("Bond_A-Snapshot_5", int64(1)),
	)

	m := bals.InterpretUsingSpecialPrefix("Bond_A", SnapshotPrefix)
	require.Len(m, 3)
	require.Equal(int64(1), m[1])
	require.Equal(int64(4), m[4])
	require.Equal(int64(-1), m[5])
}
