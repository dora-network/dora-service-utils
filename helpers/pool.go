package helpers

import (
	"github.com/dora-network/bond-api-golang/match/types"
)

// PoolTracker stores a pool
type PoolTracker struct {
	pool *types.Pool
}

func NewPoolTracker(p *types.Pool) PoolTracker {
	return PoolTracker{pool: p}
}

func (pt PoolTracker) Pool() *types.Pool {
	return pt.pool
}

func (pt PoolTracker) IsEmpty() bool {
	return pt.pool.AmountBase() == 0 || pt.pool.AmountQuote() == 0
}

/*
// EstimateMarketPrice based on the balIn passed, if it's nil, uses 1% of quote liquidity for the estimation.
func (pt PoolTracker) EstimateMarketPrice(balIn *ltypes.Balance) (float64, error) {
	if balIn == nil || balIn.IsZero() {
		balIn = ltypes.NewBalance(pt.Pool().QuoteAsset(), 1+pt.Pool().AmountQuote()/100)
	}

	return pt.Pool().SimulateSwapPrice(balIn)
}
*/
