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
		"Bond_A-USD",        // pool share
		"Bond_A-Coupon_123", // not an asset, but a valid Position.InterestSources entry
	}

	invalidAssetIDs := []string{
		"",
		"A&B Co.",           // only alphanumeric characters and underscores allowed, plus up to one hyphen
		"-abc",              // no leading hyphens
		"abc-",              // no trailing hyphens
		"Bond_A-USD-Bond_B", // a single hyphen indicates pool share. multiple are forbidden
		"Coupon_123-Bond_B", // coupon period must be after the hyphen for Position.InterestSources entry
	}

	for _, id := range validAssetIDs {
		require.Nil(ValidAssetID(id), id)
	}

	for _, id := range invalidAssetIDs {
		require.Error(ValidAssetID(id), id)
	}
}
