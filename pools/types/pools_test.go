package types_test

import (
	"testing"
	"time"

	ltypes "github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/pools/types"
	"github.com/dora-network/dora-service-utils/testing/consts"
	"github.com/govalues/decimal"
	"github.com/stretchr/testify/require"
)

func TestPool_Misc(t *testing.T) {
	require := require.New(t)
	pool := &types.Pool{
		PoolID:        consts.BondID + "-" + consts.StableID,
		BaseAsset:     consts.BondID,
		AmountBase:    0,
		QuoteAsset:    consts.StableID,
		AmountQuote:   100,
		IsProductPool: true,
		AmountShares:  200,
		CreatedAt:     1702433830,
	}

	// Add and sub assets
	addAmount := ltypes.NewAmount(consts.StableID, 100)
	require.NoError(pool.AddAmount(addAmount))
	require.NoError(pool.SubAmount(addAmount))

	// Other asset ID
	otherAsset, err := pool.OtherAssetID(consts.StableID)
	require.NoError(err)
	require.Equal(consts.BondID, otherAsset)
	otherAsset, err = pool.OtherAssetID(consts.BondID)
	require.NoError(err)
	require.Equal(consts.StableID, otherAsset)
	_, err = pool.OtherAssetID("")
	require.Error(err)
	_, err = pool.OtherAssetID(pool.PoolID)
	require.Error(err)
}

func TestAddLiquidity(t *testing.T) {
	poolID := consts.BondID + "-" + consts.StableID
	p := types.Pool{
		PoolID:        poolID,
		IsProductPool: true,
		BaseAsset:     consts.BondID,
		AmountBase:    10,
		QuoteAsset:    consts.StableID,
		AmountQuote:   10_000,
		AmountShares:  10_010,
	}

	addBond := ltypes.NewAmount(consts.BondID, 1)
	addStable := ltypes.NewAmount(consts.StableID, 1000)

	quoteAmt, sharesAmt, err := p.AddLiquidity(addBond)
	require.NoError(t, err)
	require.Equal(t, uint64(1001), sharesAmt.Amount)
	require.Equal(t, uint64(1000), quoteAmt.Amount)
	require.Equal(t, uint64(11), p.AmountBase)
	require.Equal(t, uint64(11000), p.AmountQuote)
	require.Equal(t, uint64(11011), p.AmountShares)

	_, _, err = p.AddLiquidity(addStable)
	require.Error(t, err)
}

func TestAddLiquidityFreshPool(t *testing.T) {
	poolID := consts.BondID + "-" + consts.StableID
	p := types.Pool{
		PoolID:             poolID,
		IsProductPool:      true,
		BaseAsset:          consts.BondID,
		AmountBase:         0,
		QuoteAsset:         consts.StableID,
		AmountQuote:        0,
		AmountShares:       0,
		InitialAssetsRatio: decimal.MustParse("0.8"),
		Sequence:           3,
	}

	addBond := ltypes.NewAmount(consts.BondID, 1000)
	quoteAmt, sharesAmt, err := p.AddLiquidity(addBond)
	require.NoError(t, err)
	require.Equal(t, uint64(1800), sharesAmt.Amount)
	require.Equal(t, uint64(800), quoteAmt.Amount)
	require.Equal(t, uint64(1000), p.AmountBase)
	require.Equal(t, uint64(800), p.AmountQuote)
	require.Equal(t, uint64(1800), p.AmountShares)
	require.Equal(t, uint64(4), p.Sequence)
}

func TestRemoveLiquidity(t *testing.T) {
	poolID := consts.BondID + "-" + consts.StableID
	p := types.Pool{
		PoolID:        poolID,
		IsProductPool: true,
		BaseAsset:     consts.BondID,
		AmountBase:    10,
		QuoteAsset:    consts.StableID,
		AmountQuote:   10_000,
		AmountShares:  10_010,
		Sequence:      3,
	}

	sharesToRemove := ltypes.NewAmount(poolID, 3003)
	assetsOut, err := p.RemoveLiquidity(sharesToRemove)
	require.NoError(t, err)
	require.Equal(t, consts.BondID, assetsOut[0].AssetID)
	require.Equal(t, uint64(3), assetsOut[0].Amount)
	require.Equal(t, consts.StableID, assetsOut[1].AssetID)
	require.Equal(t, uint64(3000), assetsOut[1].Amount)

	require.Equal(t, uint64(7007), p.AmountShares)

	require.Equal(t, uint64(7), p.AmountBase)
	require.Equal(t, uint64(7000), p.AmountQuote)
	require.Equal(t, uint64(4), p.Sequence)
}

func TestPoolPrice(t *testing.T) {
	poolID := consts.BondID + "-" + consts.StableID
	p := types.Pool{
		PoolID:      poolID,
		BaseAsset:   consts.BondID,
		AmountBase:  10_000,
		QuoteAsset:  consts.StableID,
		AmountQuote: 10_000,
		CreatedAt:   time.Now().Add(-24 * time.Hour).Unix(),
		MaturityAt:  time.Now().Add(24 * time.Hour).Unix(),
		// FeeFactor: 0 = no fee for product pool
		FeeFactor:     decimal.Zero,
		IsProductPool: true,
	}

	price, err := p.Price()
	require.NoError(t, err)
	require.Equal(t, float64(1.000050505050505), price)

	// FeeFactor: 1 = no fee for product pool
	p.IsProductPool = false
	p.FeeFactor = decimal.One

	price, err = p.Price()
	require.NoError(t, err)
	require.Equal(t, float64(1.000050505050505), price)

	tParam, err := p.T()
	require.NoError(t, err)
	require.Equal(t, 0.5, tParam)
}
