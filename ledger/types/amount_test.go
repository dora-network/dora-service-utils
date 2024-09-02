package types_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/testing/consts"
)

func TestAmount_Add(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		title  string
		init   types.Amount
		add    types.Amount
		errMsg string
		result types.Amount
	}{
		{
			title:  "diff assetID",
			init:   types.ZeroAmount(consts.BondID),
			add:    types.NewAmount(consts.StableID, 1),
			errMsg: "AssetIDs did not match",
			result: types.Amount{},
		},
		{
			title:  "overflow",
			init:   types.NewAmount(consts.StableID, math.MaxUint64),
			add:    types.NewAmount(consts.StableID, 1),
			errMsg: "overflow in addition",
			result: types.Amount{},
		},
		{
			title:  "correct",
			init:   types.NewAmount(consts.StableID, math.MaxUint64-1),
			add:    types.NewAmount(consts.StableID, 1),
			errMsg: "",
			result: types.NewAmount(consts.StableID, math.MaxUint64),
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
		init   types.Amount
		sub    types.Amount
		errMsg string
		result types.Amount
	}{
		{
			title:  "diff assetID",
			init:   types.ZeroAmount(consts.BondID),
			sub:    types.NewAmount(consts.StableID, 1),
			errMsg: "AssetIDs did not match",
			result: types.Amount{},
		},
		{
			title:  "negative overflow",
			init:   types.NewAmount(consts.StableID, 133),
			sub:    types.NewAmount(consts.StableID, 134),
			errMsg: "overflow in subtraction",
			result: types.Amount{},
		},
		{
			title:  "correct",
			init:   types.NewAmount(consts.StableID, 2),
			sub:    types.NewAmount(consts.StableID, 1),
			errMsg: "",
			result: types.NewAmount(consts.StableID, 1),
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
		init   types.Amount
		sub    types.Amount
		errMsg string
		result types.Amount
	}{
		{
			title:  "diff assetID",
			init:   types.ZeroAmount(consts.BondID),
			sub:    types.NewAmount(consts.StableID, 1),
			errMsg: "AssetIDs did not match",
			result: types.Amount{},
		},
		{
			title:  "no negative overflow",
			init:   types.NewAmount(consts.StableID, 133),
			sub:    types.NewAmount(consts.StableID, 134),
			errMsg: "",
			result: types.ZeroAmount(consts.StableID),
		},
		{
			title:  "correct",
			init:   types.NewAmount(consts.StableID, 2),
			sub:    types.NewAmount(consts.StableID, 1),
			errMsg: "",
			result: types.NewAmount(consts.StableID, 1),
		},
	}

	for _, tc := range tcs {
		t.Run(
			tc.title, func(t *testing.T) {
				result, _, err := tc.init.SubToZero(tc.sub)
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

	zero := types.ZeroAmount(consts.StableID)
	one := types.NewAmount(consts.BondID, 1)
	require.True(t, zero.IsZero())
	require.False(t, one.IsZero())
	require.True(t, zero.LTUint64(one.Amount))
	require.True(t, one.GTUint64(zero.Amount))
	require.True(t, zero.LTEUint64(one.Amount))
	require.True(t, zero.LTEUint64(zero.Amount))
	require.True(t, one.GTEUint64(zero.Amount))
	require.True(t, one.GTEUint64(one.Amount))
	require.Error(t, types.Amount{}.Validate())
}
