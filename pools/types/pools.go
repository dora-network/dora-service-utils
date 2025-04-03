package types

import (
	"fmt"
	gmath "math"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/dora-network/dora-service-utils/errors"
	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/math"
	"github.com/goccy/go-json"
	"github.com/govalues/decimal"
)

// Pool represents a liquidity pool in the DORA network.
// This struct is for serialization purposes only.
type Pool struct {
	PoolID             string          `json:"pool_id" redis:"pool_id"`
	BaseAsset          string          `json:"base_asset" redis:"base_asset"`
	QuoteAsset         string          `json:"quote_asset" redis:"quote_asset"`
	IsProductPool      bool            `json:"is_product_pool" redis:"is_product_pool"`
	AmountShares       uint64          `json:"amount_shares" redis:"amount_shares"`
	AmountBase         uint64          `json:"amount_base" redis:"amount_base"`
	AmountQuote        uint64          `json:"amount_quote" redis:"amount_quote"`
	FeesCollectedBase  uint64          `json:"fees_collected_base" redis:"fees_collected_base"`
	FeesCollectedQuote uint64          `json:"fees_collected_quote" redis:"fees_collected_quote"`
	FeeFactor          decimal.Decimal `json:"fee_factor" redis:"fee_factor"`
	CreatedAt          int64           `json:"created_at" redis:"created_at"`
	MaturityAt         int64           `json:"maturity_at" redis:"maturity_at"`
	InitialAssetsRatio decimal.Decimal `json:"initial_assets_ratio" redis:"initial_assets_ratio"`
	DisplayName        string          `json:"display_name" redis:"display_name"`
}

func (p *Pool) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Pool) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}

func (p *Pool) GetFeesCollectedBase() uint64 {
	return p.FeesCollectedBase
}

func (p *Pool) GetFeesCollectedQuote() uint64 {
	return p.FeesCollectedQuote
}

// AddAccumulatedFees to a pool. Error on invalid balance or asset ID not matching pool's base or quote.
func (p *Pool) AddAccumulatedFees(bal *types.Balance) error {
	if !bal.Valid() {
		return errors.New(errors.InvalidInputError, "invalid object")
	}
	switch bal.Asset {
	case p.BaseAsset:
		p.FeesCollectedBase += bal.Amount
		return nil
	case p.QuoteAsset:
		p.FeesCollectedQuote += bal.Amount
		return nil
	default:
		return errors.New(errors.InvalidInputError, "did not match pool assets")
	}
}

// Amount retrieves a specific asset from pool.
func (p *Pool) Amount(assetUID string) types.Amount {
	if assetUID == p.BaseAsset {
		return types.NewAmount(p.BaseAsset, p.AmountBase)
	}
	if assetUID == p.QuoteAsset {
		return types.NewAmount(p.QuoteAsset, p.AmountQuote)
	}
	return types.Amount{}
}

// AssetIDs of the pool, in alphabetical order.
// Requires pool.Assets having all assets present (zero balances should not be omitted),
// and pools with len(assets) != 2 may behave unexpectedly downstream
func (p *Pool) AssetIDs() (ids []string) {
	ids = append(ids, p.BaseAsset, p.QuoteAsset)
	sort.Slice(
		ids, func(i, j int) bool {
			return ids[i] < ids[j]
		},
	)
	return ids
}

// Amounts Returns the two amounts of the pool, with a specified asset listed first. Error if asset not found.
func (p *Pool) Amounts(assetA string) (amountA, amountB types.Amount, err error) {
	if assetA == p.BaseAsset {
		return types.NewAmount(p.BaseAsset, p.AmountBase), types.NewAmount(p.QuoteAsset, p.AmountQuote), nil
	}
	if assetA == p.QuoteAsset {
		return types.NewAmount(p.QuoteAsset, p.AmountQuote), types.NewAmount(p.BaseAsset, p.AmountBase), nil
	}
	return types.Amount{}, types.Amount{}, errors.ErrAssetNotFoundInPool
}

// OtherAssetID returns the id of the other asset in a pool, when one is provided.
// Error if the assets are duplicate, or the asset provided is not in pool.
func (p *Pool) OtherAssetID(IDin string) (IDout string, err error) {
	ids := p.AssetIDs()
	if strings.EqualFold(ids[0], ids[1]) {
		return "", errors.ErrPoolShouldHave2Assets
	}
	switch strings.ToLower(IDin) {
	case strings.ToLower(ids[0]):
		return ids[1], nil
	case strings.ToLower(ids[1]):
		return ids[0], nil
	default:
		return "", errors.ErrAssetNotFoundInPool
	}
}

// OtherAssetIDFromSet returns the id of the other asset in a pool, when one is from a provided list.
// If neither or both pool assets are from the provided list, no ID is returned and ok is false.
func (p *Pool) OtherAssetIDFromSet(IDs []string) (IDout string, ok bool) {
	ids := p.AssetIDs()
	match0 := contains(IDs, ids[0])
	match1 := contains(IDs, ids[1])
	if match0 && match1 {
		return "", false // both pool assets were in list
	}
	if match0 {
		return ids[1], true // ids[0] was in list, return ids[1]
	}
	if match1 {
		return ids[0], true // ids[1] was in list, return ids[0]
	}
	return "", false
}

func (p *Pool) Validate() error {
	base := p.BaseAsset
	quote := p.QuoteAsset
	if HasHyphen(base) || HasHyphen(quote) {
		return errors.Data(
			"cannot create pool where one asset is a pool share (%s-%s)",
			base,
			quote,
		)
	}
	// TODO
	return nil
}

// AddAmount to the pool. Error if asset ID is not one of the pool's assets.
func (p *Pool) AddAmount(amount types.Amount) error {
	if amount.AssetID == p.BaseAsset {
		newAmount, err := amount.AddUint64(p.AmountBase)
		if err != nil {
			return err
		}
		p.AmountBase = newAmount.Amount
	} else if amount.AssetID == p.QuoteAsset {
		newAmount, err := amount.AddUint64(p.AmountQuote)
		if err != nil {
			return err
		}
		p.AmountQuote = newAmount.Amount
	} else {
		return errors.Data("AddAsset: asset %s not found in pool %s", amount.AssetID, p.PoolID)
	}
	return nil
}

// SubAmount from the pool. Error if asset ID is not one of the pool's assets, or if amount is too great.
func (p *Pool) SubAmount(amount types.Amount) error {
	if amount.AssetID == p.BaseAsset {
		newAmount, err := math.CheckedSubU64(p.AmountBase, amount.Amount)
		if err != nil {
			return err
		}
		p.AmountBase = newAmount
	} else if amount.AssetID == p.QuoteAsset {
		newAmount, err := math.CheckedSubU64(p.AmountQuote, amount.Amount)
		if err != nil {
			return err
		}
		p.AmountQuote = newAmount
	} else {
		return errors.Data("SubAmount: asset %s not found in pool %s", amount.AssetID, p.PoolID)
	}
	return nil
}

// AddLiquidity to a pool, based on the assets given. Pool is mutated.
func (p *Pool) AddLiquidity(baseIn types.Amount) (
	quoteIn types.Amount,
	sharesOut types.Amount,
	err error,
) {
	if err = baseIn.Validate(); err != nil {
		return types.Amount{}, types.Amount{}, err
	}
	if baseIn.IsZero() {
		return types.Amount{}, types.Amount{}, errors.ErrAmountCannotBeZero
	}
	if baseIn.AssetID != p.BaseAsset {
		return types.Amount{}, types.Amount{}, errors.ErrBaseAssetMismatch
	}

	// What portion of the pool is being added? For example, adding 20 to a pool of 100 results in 0.2 here
	baseInF := new(big.Float).SetInt64(int64(baseIn.Amount))
	poolBaseF := new(big.Float).SetInt64(int64(p.AmountBase))
	poolQuoteF := new(big.Float).SetInt64(int64(p.AmountQuote))
	poolSharesF := new(big.Float).SetInt64(int64(p.AmountShares))
	if math.IsFloatZero(poolBaseF) {
		// Calculate quote assets in
		ratio, _ := p.InitialAssetsRatio.Float64()
		quoteInAmtF := math.MulFloat64ByBigFloat(ratio, baseInF)
		quoteIn = types.NewAmount(p.QuoteAsset, math.Int(quoteInAmtF).Uint64())
		// Calculate shares out
		sharesOutAmt := math.Int(math.AddF(baseInF, quoteInAmtF))
		sharesOut = types.NewAmount(p.PoolID, sharesOutAmt.Uint64())
	} else {
		addedRatio := new(big.Float).Quo(baseInF, poolBaseF)
		// Calculate quote assets in
		quoteInAmt := math.Int(math.MulF(poolQuoteF, addedRatio))
		quoteIn = types.NewAmount(p.QuoteAsset, quoteInAmt.Uint64())
		// Calculate shares out
		sharesOutAmt := math.Int(math.MulF(poolSharesF, addedRatio))
		sharesOut = types.NewAmount(p.PoolID, sharesOutAmt.Uint64())
	}

	// Mutate the pool
	if p.AmountShares, err = math.CheckedAddU64(p.AmountShares, sharesOut.Amount); err != nil {
		return types.Amount{}, types.Amount{}, err
	}
	if p.AmountBase, err = math.CheckedAddU64(p.AmountBase, baseIn.Amount); err != nil {
		return types.Amount{}, types.Amount{}, err
	}
	if p.AmountQuote, err = math.CheckedAddU64(p.AmountQuote, quoteIn.Amount); err != nil {
		return types.Amount{}, types.Amount{}, err
	}
	return quoteIn, sharesOut, nil
}

// RemoveLiquidity from a pool, based on the shares given. Pool is mutated.
func (p *Pool) RemoveLiquidity(sharesIn types.Amount) ([]types.Amount, error) {
	if err := sharesIn.Validate(); err != nil {
		return nil, err
	}
	if sharesIn.IsZero() {
		return nil, errors.ErrAmountCannotBeZero
	}
	if sharesIn.AssetID != p.PoolID {
		return nil, fmt.Errorf("RemoveLiquidity: pool and lp asset id mismatch")
	}
	if p.AmountShares < sharesIn.Amount {
		return nil, fmt.Errorf("RemoveLiquidity: removing more sharesIn than pool has")
	}

	// What portion of the pool is being withdrawn (for example, 14 out of 50 sharesIn would be 0.28)
	sharesF := new(big.Float).SetInt64(int64(sharesIn.Amount))
	poolBaseF := new(big.Float).SetInt64(int64(p.AmountBase))
	poolQuoteF := new(big.Float).SetInt64(int64(p.AmountQuote))
	poolSharesF := new(big.Float).SetInt64(int64(p.AmountShares))
	removeRatio := new(big.Float).Quo(sharesF, poolSharesF)
	// Calculate amounts out
	baseOutAmt := math.Int(math.MulF(poolBaseF, removeRatio))
	quoteOutAmt := math.Int(math.MulF(poolQuoteF, removeRatio))
	baseOut := types.NewAmount(p.BaseAsset, baseOutAmt.Uint64())
	quoteOut := types.NewAmount(p.QuoteAsset, quoteOutAmt.Uint64())

	// Mutate the pool
	var err error
	if p.AmountShares, err = math.CheckedSubU64(p.AmountShares, sharesIn.Amount); err != nil {
		return nil, err
	}
	if p.AmountBase, err = math.CheckedSubU64(p.AmountBase, baseOut.Amount); err != nil {
		return nil, err
	}
	if p.AmountQuote, err = math.CheckedSubU64(p.AmountQuote, quoteOut.Amount); err != nil {
		return nil, err
	}

	return []types.Amount{baseOut, quoteOut}, nil
}

// contains returns true if a slice of strings contains a specified string (case insensitive)
func contains(set []string, target string) bool {
	for _, s := range set {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}

func HasHyphen(UID string) bool {
	return strings.Contains(UID, "-")
}

func (p *Pool) Balances(assetA string) (balanceA, balanceB *types.Balance, err error) {
	if assetA == p.BaseAsset {
		return types.NewBalance(p.BaseAsset, int64(p.AmountBase)), types.NewBalance(
			p.QuoteAsset,
			int64(p.AmountQuote),
		), err
	}

	if assetA == p.QuoteAsset {
		return types.NewBalance(p.QuoteAsset, int64(p.AmountQuote)), types.NewBalance(
			p.BaseAsset,
			int64(p.AmountBase),
		), err
	}

	err = errors.ErrAssetNotFoundInPool

	return
}

func (p *Pool) SimulateSwap(balanceIn *types.Balance) (balanceOut, balanceFee *types.Balance, err error) {
	if p.IsProductPool {
		return p.simulateProductPoolSwap(balanceIn)
	}

	return p.simulateYieldPoolSwap(balanceIn)
}

func (p *Pool) simulateProductPoolSwap(balanceIn *types.Balance) (*types.Balance, *types.Balance, error) {
	if !balanceIn.Valid() {
		return nil, nil, errors.New(errors.InvalidInputError, "invalid balance")
	}
	if balanceIn.IsZero() {
		return nil, nil, errors.ErrAmountCannotBeZero
	}
	poolReserveAssetIn, poolReserveAssetOut, err := p.Balances(balanceIn.Asset)
	if err != nil {
		return nil, nil, err
	}

	amtIn := big.NewInt(balanceIn.Amt())
	fee := math.MulIF64(amtIn, p.SwapFee())
	amtInAfterFee := math.Sub(amtIn, fee)
	if amtInAfterFee.Sign() != 1 {
		return nil, nil, errors.ErrInsufficientBalance
	}

	poolReserveAmountOut := big.NewInt(int64(poolReserveAssetOut.Amount))
	poolReserveAmountIn := big.NewInt(int64(poolReserveAssetIn.Amount))

	poolReserveAmountOutMulAmountIn := math.Mul(poolReserveAmountOut, amtInAfterFee)
	poolReserveAmountInPlusAmountIn := math.Add(poolReserveAmountIn, amtInAfterFee)
	amountOut := new(big.Int).Quo(poolReserveAmountOutMulAmountIn, poolReserveAmountInPlusAmountIn)

	balanceOut := types.NewBalance(poolReserveAssetOut.Asset, amountOut.Int64())
	balanceFee := types.NewBalance(balanceIn.Asset, fee.Int64())

	return balanceOut, balanceFee, nil
}

// calculate t, which should range from 1 to 0 over the life of the bond,
// derived from the current time (now), maturity (end) time, and total duration of the bond.
// returns 0 if time is past maturity (that is, now >= end). Also returns zero on invalid duration.
func calculateT(now, end, duration int64) float64 {
	if duration <= 0 || now >= end {
		return 0
	}
	t := float64(end-now) / float64(duration)
	if t > 1 {
		t = 1
	}
	return t
}

// nolint:unused
// Calculates z := 1-tg for buys, or z := 1-t/g for sells.
// G should never be zero due to pool validation (though we will return 0 rather than panic if it is)
// If the result z is zero, special action will need to be taken in downstream swap math. This will occur once
// in every pool's lifetime for sells and cause them to behave like a product pool for sell swaps at that unix time.
// Also applies to buys for pools with zero fees for swaps occurring at the same unix time they were created.
func calculateZ(t, g float64, isSell bool) float64 {
	if g == 0 {
		return 0 // Buy or Sell with invalid G
	}
	if isSell {
		return 1 - (t / g) // Swap is a Sell
	}
	return 1 - t*g // Swap is a Buy
}

// returns k := x^t' + y^t'
// where t' := 1 - (t * g) and t progresses from 1 to 0 over time
func calculateK(x, y *big.Int, t, g float64) float64 {
	xf, _ := x.Float64()
	yf, _ := y.Float64()
	tPrime := 1 - (t * g)
	return gmath.Pow(xf, tPrime) + gmath.Pow(yf, tPrime)
}

// Calculates how the out asset changes if we increase in asset.
// <param:assetInBalance> [x_end] = x_start + ∆x
//
// y_end = [k - x_end^(1-tg)]^[1/(1-t)]  || Note: tg can also be t/g depending on the asset.
//
// ∆y = y_end - y_start
func calculateDelta(
	assetOutBalance, assetInBalance *big.Int,
	k, timeToMaturity, gFeeFactor float64,
) *big.Int {
	in, _ := assetInBalance.Float64()
	out, _ := assetOutBalance.Float64()

	updatedOut := k - gmath.Pow(in, 1-(gFeeFactor*timeToMaturity))
	updatedOut = gmath.Pow(updatedOut, 1/(1-(gFeeFactor*timeToMaturity)))
	delta := math.Int(big.NewFloat(updatedOut - out))
	return delta
}

func (p *Pool) Duration() int64 {
	if p.IsProductPool {
		return 0
	}
	return p.MaturityAt - p.CreatedAt
}

func (p *Pool) SwapFee() float64 {
	if !p.IsProductPool {
		return 0
	}

	ff, _ := p.FeeFactor.Float64()
	return ff
}

func (p *Pool) G() float64 {
	if p.IsProductPool {
		return 0
	}

	ff, _ := p.FeeFactor.Float64()
	return ff
}

func (p *Pool) simulateYieldPoolSwap(balanceIn *types.Balance) (*types.Balance, *types.Balance, error) {
	if !balanceIn.Valid() {
		return nil, nil, errors.New(errors.InvalidInputError, "invalid balance")
	}
	if balanceIn.IsZero() {
		return nil, nil, errors.ErrAmountCannotBeZero
	}
	timeToMaturity := calculateT(time.Now().Unix(), p.MaturityAt, p.Duration())

	stableAmount := types.Balance{
		Asset:  p.QuoteAsset,
		Amount: p.AmountQuote,
	}

	bondAmount := types.Balance{
		Asset:  p.BaseAsset,
		Amount: p.AmountBase,
	}

	assetOutID, err := p.OtherAssetID(balanceIn.Asset)
	if err != nil {
		return nil, nil, err
	}

	poolBalanceOfIn, poolBalanceOfOut, err := p.Balances(balanceIn.Asset)
	if err != nil {
		return nil, nil, err
	}

	G := p.G()
	if balanceIn.Asset == p.BaseAsset {
		G = 1 / G
	}

	poolBalanceOfIn, err = poolBalanceOfIn.Add(balanceIn)
	if err != nil {
		return nil, nil, err
	}

	k := calculateK(
		big.NewInt(int64(stableAmount.Amount)),
		big.NewInt(int64(bondAmount.Amount)),
		timeToMaturity,
		1,
	)
	pbOut := big.NewInt(int64(poolBalanceOfOut.Amount))
	pbIn := big.NewInt(int64(poolBalanceOfIn.Amount))
	outWithoutFee := calculateDelta(
		pbOut,
		pbIn,
		k,
		timeToMaturity,
		1,
	)
	k = calculateK(
		big.NewInt(int64(stableAmount.Amount)),
		big.NewInt(int64(bondAmount.Amount)),
		timeToMaturity,
		G,
	)
	outWithFee := calculateDelta(
		pbOut,
		pbIn,
		k,
		timeToMaturity,
		G,
	)

	feeAmt := math.Sub(outWithFee, outWithoutFee)
	outWithFee = math.Mul(outWithFee, big.NewInt(-1))
	balanceOut := types.NewBalance(assetOutID, outWithFee.Int64())
	balanceFee := types.NewBalance(balanceIn.Asset, feeAmt.Int64())

	return balanceOut, balanceFee, nil
}

func (p *Pool) SimulateSwapPrice(isSell bool, balIn *types.Balance) (float64, error) {
	var (
		err    error
		balOut *types.Balance
	)

	balOut, _, err = p.SimulateSwap(balIn)
	if err != nil {
		return 0, err
	}

	return math.ExecutedPrice(
		isSell,
		balIn.Amount,
		balOut.Amount,
	), nil
}
