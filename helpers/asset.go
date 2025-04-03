package helpers

import (
	"errors"
	"fmt"
	"math"

	"github.com/dora-network/dora-service-utils/ledger/types"
	smath "github.com/dora-network/dora-service-utils/math"
)

// AssetData holds asset info and prices used for matchmaking
type AssetData struct {
	// asset parameters (only as needed by matchmaking)
	decimals              map[string]int
	collateralWeights     map[string]float64
	liquidationThresholds map[string]float64

	// usage flags
	isCurrency      map[string]bool
	isBond          map[string]bool
	canTrade        map[string]bool
	canBorrow       map[string]bool
	isVirtualBorrow map[string]bool
	isInterest      map[string]bool

	// dirty price AMM
	coupons map[string][]Coupon

	// also track asset prices
	prices map[string]float64

	// internal
	initialized bool
}

func (ad *AssetData) Init() {
	if !ad.initialized {
		// maps need to be set to non-nil
		ad.decimals = map[string]int{}
		ad.collateralWeights = map[string]float64{}
		ad.liquidationThresholds = map[string]float64{}

		ad.isCurrency = map[string]bool{}
		ad.isBond = map[string]bool{}
		ad.canTrade = map[string]bool{}
		ad.canBorrow = map[string]bool{}
		ad.isVirtualBorrow = map[string]bool{}
		ad.isInterest = map[string]bool{}

		ad.coupons = map[string][]Coupon{}

		ad.prices = map[string]float64{}

		ad.initialized = true
	}
}

// RegisterAsset stores all of an asset's relevant attributes for matchmaking. Does not require asset price.
func (ad *AssetData) RegisterAsset(
	id string,

	decimals int,
	collateralWeight,
	liquidationThreshold float64,

	isCurrency,
	isBond,
	canTrade,
	canBorrow,
	isVirtualBorrow,
	isInterest bool,
	coupons []*Coupon,
) error {
	ad.Init()
	// Validate asset data
	if id == "" {
		return errors.New("empty asset ID")
	}
	if math.IsNaN(collateralWeight) || math.IsInf(collateralWeight, 0) || collateralWeight < 0 {
		return errors.New("invalid collateral weight")
	}
	if math.IsNaN(liquidationThreshold) || math.IsInf(liquidationThreshold, 0) || liquidationThreshold < 0 {
		return errors.New("invalid liquidation threshold")
	}
	if decimals < 0 {
		return errors.New("invalid exponent")
	}
	if isVirtualBorrow && !canBorrow {
		return errors.New("asset with isVirtualBorrow but without canBorrow is not valid")
	}
	// All params are valid: set in AssetData
	ad.decimals[id] = decimals
	ad.collateralWeights[id] = collateralWeight
	ad.liquidationThresholds[id] = liquidationThreshold

	ad.isCurrency[id] = isCurrency
	ad.isBond[id] = isBond
	ad.canTrade[id] = canTrade
	ad.canBorrow[id] = canBorrow
	ad.isVirtualBorrow[id] = isVirtualBorrow
	ad.isInterest[id] = isInterest

	ad.coupons[id] = []Coupon{}
	for _, c := range coupons {
		if c != nil && c.Yield > 0 {
			ad.coupons[id] = append(ad.coupons[id], *c)
		}
	}

	return nil
}

func (ad *AssetData) UpdatePrice(
	id string,
	price float64,
) error {
	ad.Init()
	if id == "" {
		return errors.New("empty asset ID")
	}
	if math.IsNaN(price) || math.IsInf(price, 0) || price < 0 {
		return errors.New("invalid price")
	}
	ad.prices[id] = price
	return nil
}

// IDs returns an unsorted slice of the assetIDs of all registered assets
func (ad *AssetData) IDs() []string {
	ids := []string{}
	for id := range ad.decimals {
		ids = append(ids, id)
	}
	return ids
}

// Has returns true if AssetData has a non-nil asset with given ID
func (ad AssetData) Has(assetID string) bool {
	if ad.decimals == nil {
		return false
	}
	_, ok := ad.decimals[assetID]
	return ok
}

// Decimals
func (ad AssetData) Decimals(assetID string) (int, error) {
	if ad.decimals == nil {
		return 0, fmt.Errorf("asset %s not found", assetID)
	}
	d, ok := ad.decimals[assetID]
	if !ok {
		return 0, fmt.Errorf("asset %s not found", assetID)
	}
	return d, nil
}

// CouponEndingAt returns an asset's coupon period which ends at time. Error if no such period, or multiple.
// Returns asset maturity payment if the input time is the maturity date.
func (ad AssetData) CouponEndingAt(assetID string, time int64) (*Coupon, error) {
	if ad.coupons == nil {
		err := fmt.Errorf("asset %s not found", assetID)
		return nil, err
	}
	result := []Coupon{}
	for _, c := range ad.coupons[assetID] {
		end, err := smath.UnixFromDate(c.Date)
		if err != nil {
			return nil, err
		}
		if end == time && c.Yield > 0 {
			result = append(result, c)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("coupon period for %s ending at %d not found", assetID, time)
	}
	if len(result) != 1 {
		return nil, fmt.Errorf("multiple coupon periods for %s ending at %d found", assetID, time)
	}
	// only return if exactly one match
	return &result[0], nil
}

// CouponsEndingInRange returns any of an asset's coupon periods which end after fromTime
// AND (before OR at) toTime. (If the two times are equal, the coupon period is not returned.)
// Also ignores coupon periods without a positive yield.
// Returns asset maturity payment as well if it occurs in the date range.
func (ad AssetData) CouponsEndingInRange(assetID string, fromTime, toTime int64) ([]Coupon, error) {
	if ad.coupons == nil {
		err := fmt.Errorf("asset %s not found", assetID)
		return nil, err
	}
	result := []Coupon{}
	for _, c := range ad.coupons[assetID] {
		end, err := smath.UnixFromDate(c.Date)
		if err != nil {
			return nil, err
		}
		if fromTime < end && end <= toTime && c.Yield > 0 {
			result = append(result, c)
		}
	}
	return result, nil
}

// CurrentCouponPeriod returns the in-progress coupon period of an asset, if there is one at a given time.
func (ad AssetData) CurrentCouponPeriod(assetID string, now int64) (start, end int64, yield float64, err error) {
	if ad.coupons == nil {
		err = fmt.Errorf("asset %s not found", assetID)
		return 0, 0, 0.0, err
	}
	coupons, ok := ad.coupons[assetID]
	if !ok || coupons == nil {
		return 0, 0, 0.0, nil // no error, but no coupons found
	}
	for _, c := range coupons {
		if c.Start == "" {
			continue // instant payments have start date empty, and are never considered "in-progress"
		}
		start, err = smath.UnixFromDate(c.Start)
		if err != nil {
			return 0, 0, 0.0, err
		}
		end, err = smath.UnixFromDate(c.Date)
		if err != nil {
			return 0, 0, 0.0, err
		}
		if start <= now && end > now {
			// return the current coupon period.
			// As our rule, the second a coupon period ends is NOT considered part of its period
			return start, end, c.Yield, nil
		}
	}
	return 0, 0, 0.0, err // no error, but no current period
}

// IsCurrency returns true if AssetData has a non-nil asset with given ID, and that asset has AssetTypeCurrency
func (ad AssetData) IsCurrency(assetID string) bool {
	return ad.isCurrency != nil && ad.isCurrency[assetID]
}

// IsBond returns true if AssetData has a non-nil asset with given ID, and that asset has AssetTypeBond
func (ad AssetData) IsBond(assetID string) bool {
	return ad.isBond != nil && ad.isBond[assetID]
}

// IsInterest returns true if AssetData has a non-nil asset with given ID, and that asset has AssetTypeInterest
func (ad AssetData) IsInterest(assetID string) bool {
	return ad.isInterest != nil && ad.isInterest[assetID]
}

// HasCoupon returns true if AssetData has a non-nil asset with given ID, and that asset has a coupon payment
func (ad AssetData) HasCoupon(assetID string) bool {
	return len(ad.coupons) > 0
}

// CanBorrow returns true if AssetData has a non-nil asset with given ID, and that asset can be borrowed
func (ad AssetData) CanBorrow(assetID string) bool {
	return ad.canBorrow != nil && ad.canBorrow[assetID]
}

// CanTrade returns true if AssetData has a non-nil asset with given ID, and that asset can be traded
func (ad AssetData) CanTrade(assetID string) bool {
	return ad.canTrade != nil && ad.canTrade[assetID]
}

// IsVirtualBorrow returns true if AssetData has a non-nil asset with given ID, and that asset can be virtualborrowed
func (ad AssetData) IsVirtualBorrow(assetID string) bool {
	return ad.isVirtualBorrow != nil && ad.isVirtualBorrow[assetID]
}

// CollateralWeight returns an asset's collateral weight. Error if asset not registered.
func (ad AssetData) CollateralWeight(assetID string) (float64, error) {
	if ad.collateralWeights == nil {
		return 0, errors.New("assetData not initialized")
	}
	lt, ok := ad.collateralWeights[assetID]
	if ok {
		return lt, nil
	}
	return 0, fmt.Errorf("asset %s not found", assetID)
}

// LiquidationThreshold returns an asset's liquidation threshold. Error if asset not registered.
func (ad AssetData) LiquidationThreshold(assetID string) (float64, error) {
	if ad.liquidationThresholds == nil {
		return 0, errors.New("assetData not initialized")
	}
	lt, ok := ad.liquidationThresholds[assetID]
	if ok {
		return lt, nil
	}
	return 0, fmt.Errorf("asset %s not found", assetID)
}

// Price returns an asset's price. Error if asset price not registered.
func (ad AssetData) Price(assetID string) (float64, error) {
	if ad.prices == nil {
		return 0, errors.New("assetData not initialized")
	}
	p, ok := ad.prices[assetID]
	if ok {
		return p, nil
	}
	return 0, fmt.Errorf("price for %s not found", assetID)
}

// ExactLiquidationThreshold returns a positions's Liquidation Threshold.
// Only the position's positive Owned assets are considered. Missing prices result in errors.
func (ad AssetData) ExactLiquidationThreshold(p *types.Position) (float64, error) {
	total := 0.0
	err := p.Owned.Iterate(
		func(assetID string, amt int64) error {
			// Positive owned assets only
			if amt > 0 {
				lt, err := ad.LiquidationThreshold(assetID)
				if err != nil {
					return err
				}
				if lt == 0 {
					return nil // assets not meant for collateral don't need prices below
				}
				value, err := ad.GetAssetValueInUSD(amt, assetID, lt)
				if err != nil {
					return err
				}
				total += value
			}
			return nil
		},
	)
	return total, err
}

// ExactBorrowLimit returns a positions's Borrow Limit.
// Only the position's positive Owned assets are considered. Missing prices result in errors.
func (ad AssetData) ExactBorrowLimit(p *types.Position) (float64, error) {
	total := 0.0
	err := p.Owned.Iterate(
		func(assetID string, amt int64) error {
			// Positive owned assets only
			if amt > 0 {
				cw, err := ad.CollateralWeight(assetID)
				if err != nil {
					return err
				}
				// Asset CW
				if cw == 0 {
					return nil // assets not meant for collateral don't need prices below
				}
				value, err := ad.GetAssetValueInUSD(amt, assetID, cw)
				if err != nil {
					return err
				}
				total += value
			}
			return nil
		},
	)
	return total, err
}

// ExactBorrowedValue returns a positions's Borrowed Value.
// Only the position's negative Owned assets are considered. Missing prices result in errors.
func (ad AssetData) ExactBorrowedValue(p *types.Position) (float64, error) {
	total := 0.0
	err := p.Owned.Iterate(
		func(assetID string, amt int64) error {
			// Negative owned assets only
			if amt < 0 {
				amt = -1 * amt
				value, err := ad.GetAssetValueInUSD(amt, assetID, 1)
				if err != nil {
					return err
				}
				total += value
			}
			return nil
		},
	)
	return total, err
}

// ExactCollateralValue returns a positions's Collateral Value.
// Only the position's positive Owned assets with CW > 0 are considered.
// Missing prices result in errors.
func (ad AssetData) ExactCollateralValue(p *types.Position) (float64, error) {
	total := 0.0
	err := p.Owned.Iterate(
		func(assetID string, amt int64) error {
			// Positive owned assets only
			if amt > 0 {
				cw, err := ad.CollateralWeight(assetID)
				if err != nil {
					return err
				}
				if cw == 0 {
					return nil // assets not meant for collateral are not considered
				}
				value, err := ad.GetAssetValueInUSD(amt, assetID, 1)
				if err != nil {
					return err
				}
				total += value
			}
			return nil
		},
	)
	return total, err
}

// ExactSuppliedValue returns a positions's Supplied Value.
// Only the position's positive Supplied assets are considered. Missing prices result in errors.
func (ad AssetData) ExactSuppliedValue(p *types.Position) (float64, error) {
	total := 0.0
	err := p.Supplied.Iterate(
		func(assetID string, amt int64) error {
			// Positive supplied assets only
			if amt > 0 {
				value, err := ad.GetAssetValueInUSD(amt, assetID, 1)
				if err != nil {
					return err
				}
				total += value
			}
			return nil
		},
	)
	return total, err
}

// ExactAvailableValue returns a positions's Available Value.
// It will be subtract the owned and locked balance
func (ad AssetData) ExactAvailableValue(p *types.Position) (float64, error) {
	locked := 0.0
	owned := 0.0
	err := p.Locked.Iterate(
		func(assetID string, amt int64) error {
			// Positive locked assets only
			if amt > 0 {
				if !ad.isBond[assetID] {
					return nil
				}
				value, err := ad.GetAssetValueInUSD(amt, assetID, 1)
				if err != nil {
					return err
				}
				locked += value
			}
			return nil
		},
	)
	if err != nil {
		return locked, err
	}
	err = p.Owned.Iterate(
		func(assetID string, amt int64) error {
			// Positive Owned assets only
			if amt > 0 {
				if !ad.isBond[assetID] {
					return nil
				}
				value, err := ad.GetAssetValueInUSD(amt, assetID, 1)
				if err != nil {
					return err
				}
				owned += value
			}
			return nil
		},
	)
	available := 0.0
	if owned >= locked {
		available = owned - locked
	}
	return available, err
}

// GetAssetValueInUSD will convert amount into USD.
func (ad AssetData) GetAssetValueInUSD(amt int64, assetID string, multiplier float64) (float64, error) {
	decimals, err := ad.Decimals(assetID)
	if err != nil {
		return 0.0, err
	}
	multiplier /= math.Pow10(decimals)
	// For known prices, compute exact value
	price, err := ad.Price(assetID)
	if err != nil {
		return 0.0, err
	}
	return price * float64(amt) * multiplier, nil
}
