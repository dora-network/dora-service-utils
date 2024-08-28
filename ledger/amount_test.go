package ledger_test

import (
	"github.com/dora-network/dora-service-utils/ledger"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

const (
	BondID    = "bond1"
	StableID  = "stable1"
	UserIDOne = "user1"
	UserIDTwo = "user2"
)

func TestAmount_Add(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		title  string
		init   ledger.Amount
		add    ledger.Amount
		errMsg string
		result ledger.Amount
	}{
		{
			title:  "diff assetID",
			init:   ledger.ZeroAmount(BondID),
			add:    ledger.NewAmount(StableID, 1),
			errMsg: "AssetIDs did not match",
			result: ledger.Amount{},
		},
		{
			title:  "overflow",
			init:   ledger.NewAmount(StableID, math.MaxUint64),
			add:    ledger.NewAmount(StableID, 1),
			errMsg: "overflow in addition",
			result: ledger.Amount{},
		},
		{
			title:  "correct",
			init:   ledger.NewAmount(StableID, math.MaxUint64-1),
			add:    ledger.NewAmount(StableID, 1),
			errMsg: "",
			result: ledger.NewAmount(StableID, math.MaxUint64),
		},
	}

	for _, tc := range tcs {
		t.Run(
			tc.title, func(t *testing.T) {
				result, err := tc.init.Add(tc.add)
				if len(tc.errMsg) > 0 {
					require.Error(t, err)
					require.Contains(t, err.Error(), tc.errMsg)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.result, result)
					require.True(t, tc.result.Equal(result))
				}
			},
		)
	}
}

func TestAmount_Sub(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		title  string
		init   ledger.Amount
		sub    ledger.Amount
		errMsg string
		result ledger.Amount
	}{
		{
			title:  "diff assetID",
			init:   ledger.ZeroAmount(BondID),
			sub:    ledger.NewAmount(StableID, 1),
			errMsg: "AssetIDs did not match",
			result: ledger.Amount{},
		},
		{
			title:  "negative overflow",
			init:   ledger.NewAmount(StableID, 133),
			sub:    ledger.NewAmount(StableID, 134),
			errMsg: "overflow in subtraction",
			result: ledger.Amount{},
		},
		{
			title:  "correct",
			init:   ledger.NewAmount(StableID, 2),
			sub:    ledger.NewAmount(StableID, 1),
			errMsg: "",
			result: ledger.NewAmount(StableID, 1),
		},
	}

	for _, tc := range tcs {
		t.Run(
			tc.title, func(t *testing.T) {
				result, err := tc.init.Sub(tc.sub)
				if len(tc.errMsg) > 0 {
					require.Error(t, err)
					require.Contains(t, err.Error(), tc.errMsg)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.result, result)
					require.True(t, tc.result.Equal(result))
				}
			},
		)
	}
}

func TestAmount_SubToZero(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		title  string
		init   ledger.Amount
		sub    ledger.Amount
		errMsg string
		result ledger.Amount
	}{
		{
			title:  "diff assetID",
			init:   ledger.ZeroAmount(BondID),
			sub:    ledger.NewAmount(StableID, 1),
			errMsg: "AssetIDs did not match",
			result: ledger.Amount{},
		},
		{
			title:  "no negative overflow",
			init:   ledger.NewAmount(StableID, 133),
			sub:    ledger.NewAmount(StableID, 134),
			errMsg: "",
			result: ledger.ZeroAmount(StableID),
		},
		{
			title:  "correct",
			init:   ledger.NewAmount(StableID, 2),
			sub:    ledger.NewAmount(StableID, 1),
			errMsg: "",
			result: ledger.NewAmount(StableID, 1),
		},
	}

	for _, tc := range tcs {
		t.Run(
			tc.title, func(t *testing.T) {
				result, err := tc.init.SubToZero(tc.sub)
				if len(tc.errMsg) > 0 {
					require.Error(t, err)
					require.Contains(t, err.Error(), tc.errMsg)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.result, result)
					require.True(t, tc.result.Equal(result))
				}
			},
		)
	}
}

func TestAmount_Misc(t *testing.T) {
	t.Parallel()
	zero := ledger.ZeroAmount(StableID)
	one := ledger.NewAmount(BondID, 1)
	require.True(t, zero.IsZero())
	require.False(t, one.IsZero())
	require.True(t, zero.LT(one))
	require.True(t, one.GT(zero))
	require.True(t, zero.LTE(one))
	require.True(t, zero.LTE(zero))
	require.True(t, one.GTE(zero))
	require.True(t, one.GTE(one))
	require.Error(t, ledger.Amount{}.Validate())
}
