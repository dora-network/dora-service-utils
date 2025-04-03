package helpers

import (
	"errors"
	"fmt"
	"math"

	"github.com/dora-network/bond-api-golang/graph/types"

	ltypes "github.com/dora-network/dora-service-utils/ledger/types"
	smath "github.com/dora-network/dora-service-utils/math"
)

// UserPositionTracker caches and records changes to user balances and borrowed,
// and exposes methods for dealing with them.
type UserPositionTracker struct {
	userPositions map[string]*ltypes.Position
}

func (ubt *UserPositionTracker) init() {
	if ubt.userPositions == nil {
		ubt.userPositions = map[string]*ltypes.Position{}
	}
}

// InitUserPosition records a single user's balances. Error if user already initialized.
func (ubt *UserPositionTracker) InitUserPosition(user string, position *ltypes.Position) error {
	ubt.init()
	if _, ok := ubt.userPositions[user]; ok {
		return fmt.Errorf("user %s already initialized", user)
	}
	// Initialize (copied balances start with modified = false)
	ubt.userPositions[user] = position.Snapshot()
	return nil
}

// modifyUserBalances adds and subtracts from a single user's balance. Error if user not initialized.
// Also error if any result would be negative. Balance tracker is not mutated on error.
func (ubt *UserPositionTracker) modifyUserBalances(user string, add, remove *ltypes.Balance) error {
	ubt.init()
	if _, ok := ubt.userPositions[user]; !ok {
		return fmt.Errorf("user %s not initialized", user)
	}
	pos := ubt.userPositions[user]
	pos.Owned = pos.Owned.Add(add).Sub(remove)
	ubt.userPositions[user] = pos
	return nil
}

// setLastUpdated
func (ubt *UserPositionTracker) setLastUpdated(user string, time int64) error {
	ubt.init()
	pos, ok := ubt.userPositions[user]
	if !ok {
		return fmt.Errorf("user %s not initialized", user)
	}
	if pos.LastUpdated > time {
		return fmt.Errorf("user %s cannot wind LastUpdated time backwards", user)
	}
	pos.LastUpdated = time
	ubt.userPositions[user] = pos
	return nil
}

// modifyUserInterestWithSource (input can be either positive or negative)
func (ubt *UserPositionTracker) modifyUserInterestWithSource(user, sourceAsset string, sourcePeriod, amount int64) error {
	ubt.init()
	if _, ok := ubt.userPositions[user]; !ok {
		return fmt.Errorf("user %s not initialized", user)
	}
	pos := ubt.userPositions[user]
	pos.InterestSources = pos.InterestSources.AddAmount(
		couponTrackerID(sourceAsset, sourcePeriod),
		amount, // can be negative
	)
	pos.Owned = pos.Owned.AddAmount("Interest", amount) // can be negative
	ubt.userPositions[user] = pos
	return nil
}

func couponTrackerID(sourceAsset string, sourcePeriod int64) string {
	return fmt.Sprintf("%s-%s%d", sourceAsset, ltypes.CouponPrefix, sourcePeriod)
}

// modifyUserBorrowed adds and subtracts from a single user's borrowed. Error if user not initialized.
// Also error if any result would be negative. Balance tracker is not mutated on error.
func (ubt *UserPositionTracker) modifyUserBorrowed(user string, add, remove *ltypes.Balance) error {
	ubt.init()
	if _, ok := ubt.userPositions[user]; !ok {
		return fmt.Errorf("user %s not initialized", user)
	}
	pos := ubt.userPositions[user]
	pos.Owned = pos.Owned.Add(remove).Sub(add) // inverted: to add a borrow is to remove a balance
	ubt.userPositions[user] = pos
	return nil
}

// Borrow adds a balance to borrowed, and to balances
func (ubt *UserPositionTracker) Borrow(user string, assetID string, amount int64) error {
	ubt.init()
	if amount < 0 {
		return errors.New("cannot borrow negative amount")
	}
	addBal := ltypes.NewBalance(assetID, amount)
	if err := ubt.modifyUserBorrowed(user, addBal, nil); err != nil {
		return err
	}
	return ubt.modifyUserBalances(user, addBal, nil)
}

// Position returns a user's most recent position
func (ubt *UserPositionTracker) Position(userID string) (*ltypes.Position, error) {
	ubt.init()
	pos, ok := ubt.userPositions[userID]
	if !ok {
		return nil, fmt.Errorf("user %s not present in position tracker", userID)
	}
	return pos, nil
}

// InitialPosition
func (ubt *UserPositionTracker) InitialPosition(userID string) string {
	ubt.init()
	return ubt.userPositions[userID].Original()
}

// FinalPosition
func (ubt *UserPositionTracker) FinalPosition(userID string) string {
	ubt.init()
	return ubt.userPositions[userID].String()
}

// UpdatedPositions returns every position that was modified, during the transactions.
func (ubt *UserPositionTracker) UpdatedPositions() map[string]*ltypes.Position {
	ubt.init()
	modified := map[string]*ltypes.Position{}
	for userID, position := range ubt.userPositions {
		if position.IsModified() {
			modified[userID] = position
		}
	}
	return modified
}

// IsLiquidationEligible detects if a user can be liquidated. If required information like asset prices
// is missing, they are not eligible and liquidation orders will be cancelled until conditions improve.
func (ubt *UserPositionTracker) IsLiquidationEligible(assets AssetData, userID string) bool {
	ubt.init()
	pos, ok := ubt.userPositions[userID]
	if !ok {
		return false
	}
	liquidationThreshold, err := assets.ExactLiquidationThreshold(pos)
	if err != nil {
		return false
	}
	borrowValue, err := assets.ExactBorrowedValue(pos)
	if err != nil {
		return false
	}
	return borrowValue > liquidationThreshold
}

// IsHealthy returns true if a user is at or below their borrow limit.
// Empty positions are considered healthy, but nonexistent users are not.
// If relevant prices cannot be computed, returns false.
func (ubt *UserPositionTracker) IsHealthy(assets AssetData, userID string) bool {
	ubt.init()
	pos, ok := ubt.userPositions[userID]
	if !ok {
		return false
	}
	borrowLimit, err := assets.ExactBorrowLimit(pos)
	if err != nil {
		return false
	}
	borrowValue, err := assets.ExactBorrowedValue(pos)
	if err != nil {
		return false
	}
	return borrowValue <= borrowLimit
}

// CanAfford tests if a user can gain, spend, and borrow certain assets and still be under
// their borrow limit. Error if user cannot afford. Also errors if any balances would be negative.
// For non-currency asset types, balances and borrows of the same asset cancel out before other calculations.
func (ubt *UserPositionTracker) CanAfford(assets AssetData, userID string, spend, borrow, gain *ltypes.Balance) error {
	ubt.init()
	p, err := ubt.Position(userID)
	if err != nil {
		return err
	}
	// Compute initial leverage values
	initialBorrowLimit, err := assets.ExactBorrowLimit(p)
	if err != nil {
		return err
	}
	initialBorrowValue, err := assets.ExactBorrowedValue(p)
	if err != nil {
		return err
	}
	// Create copies of the current balances and borrowed so the originals aren't mutated during calculations
	newPosition := p.Copy()
	// First add balances gained (automatically repays borrowed first)
	newPosition.Owned = newPosition.Owned.Add(gain)
	// Attempt to spend the required amount (moreRequired if negative)
	var moreRequired *ltypes.Balance
	newPosition.Owned, moreRequired = newPosition.Owned.SubBalToZero(spend)
	if !moreRequired.IsZero() {
		if assets.IsCurrency(moreRequired.Asset) {
			// Stablecoin equivalance will be used in this case.
			// Activating stablecoin equivalence will attempt to get balance up to the amount required
			newPosition = activateStablecoinEquivalence(assets, newPosition, moreRequired)
			// Attempt to spend the required amount and error if it still isn't present
			newPosition.Owned, err = newPosition.Owned.SafeSub(moreRequired.Asset, moreRequired.Amt())
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot afford %s %d", moreRequired.Asset, moreRequired.Amount)
		}
	}
	// Finally add any new borrows (must reduce Owned only after stablecoin equivalence activates)
	newPosition.Owned = newPosition.Owned.Sub(borrow)
	// Compute borrow leverage values and return error if borrow limit exceeded
	borrowLimit, err := assets.ExactBorrowLimit(newPosition)
	if err != nil {
		return err
	}
	borrowValue, err := assets.ExactBorrowedValue(newPosition)
	if err != nil {
		return err
	}
	if borrowValue <= borrowLimit {
		return nil // Borrower ends up healthy. This is always acceptable.
	}
	if initialBorrowValue >= initialBorrowLimit {
		// Borrower started out unhealthy and is ending up unhealthy.
		initialExcessBorrow := initialBorrowValue - initialBorrowLimit
		finalExcessBorrow := borrowValue - borrowLimit
		// In some cases, we know for sure that the result is still healthier
		if finalExcessBorrow < initialExcessBorrow && borrowValue < initialBorrowValue {
			// Both total and excess borrow have decreased
			return nil
		}
	}
	return fmt.Errorf(
		"Borrow limit %f cannot support borrowed value of %f. Collateral was %s. Borrowed was %s",
		borrowLimit, borrowValue, newPosition.Owned.Positive(), newPosition.Owned.Negative(),
	)
}

// SwapBalance subtracts BalanceIn and adds BalanceOut to a user's balance. Negative result if borrowing.
// Also attempts to unlock the amount of BalanceIn swapped.
func (ubt *UserPositionTracker) SwapBalance(
	userID string,
	balanceIn, balanceOut *ltypes.Balance,
	isDoraV1 bool,
	order *types.Order,
) error {
	position, err := ubt.Position(userID)
	if err != nil {
		return err
	}
	position.Owned = position.Owned.Sub(balanceIn)
	if !isDoraV1 || order.IsLimit() {
		position.Locked, _ = position.Locked.SubBalToZero(balanceIn)
		if order.State == types.OrderStateExecuted {
			balIn, err := order.BalanceIn()
			if err != nil {
				return err
			}
			inFilled, _, err := order.GetFilledAmts()
			if err != nil {
				return err
			}
			if balIn.Amt() > inFilled.Int64() {
				position.Locked = position.Locked.SubAmountToZero(balIn.Asset, balIn.Amt()-inFilled.Int64())
			}
		}
	}
	position.Owned = position.Owned.Add(balanceOut)
	// Apply changes
	ubt.userPositions[userID] = position
	return nil
}

// UpdateCouponInterest increases a user's interest balance due to coupon payments during the elapsed time.
// Sets position.LastUpdated to current time. Any coupon that ended AFTER position.LastUpdated and (BEFORE or AT)
// current time will disburse its payment. (If the two times are equal, no effect occurs.)
func (ubt *UserPositionTracker) UpdateCouponInterest(
	assets AssetData,
	userID string,
	// Unix time of the update
	updateUnixTime int64,
) (earned *ltypes.Balance, owed *ltypes.Balance, err error) {
	earned = ltypes.ZeroBalance("Interest")
	owed = ltypes.ZeroBalance("Interest")
	pos, err := ubt.Position(userID)
	if err != nil {
		return nil, nil, err // user position not found
	}
	lastUnixTime := pos.LastUpdated
	interestAssetExponent, err := assets.Decimals("Interest")
	if err != nil {
		return nil, nil, err // could not get asset
	}
	for _, asset := range pos.Owned.AssetIDs() {
		// Owned balance (might be negative if borrowed)
		bal := pos.Owned.AmountOf(asset)
		assetExponent, err := assets.Decimals(asset)
		if err != nil {
			return nil, nil, err // could not get asset
		}
		// Coupon periods which ended in the time elapsed since last update
		elapsedCouponPeriods, err := assets.CouponsEndingInRange(asset, lastUnixTime, updateUnixTime)
		if err != nil {
			return nil, nil, err
		}
		for _, c := range elapsedCouponPeriods {
			end, err := smath.UnixFromDate(c.Date)
			if err != nil {
				return nil, nil, err
			}
			// User earned interest because a coupon payment occurred
			couponAmountBeforeExp := c.Yield * float64(bal)
			// Apply asset exponents
			interestAmount := math.Pow10(interestAssetExponent-assetExponent) * couponAmountBeforeExp
			amt := int64(interestAmount)
			if amt > 0 {
				earned.Amount = earned.Amount + uint64(amt)
				// Apply owed interest (source is tracked by coupon period end date)
				if err = ubt.modifyUserInterestWithSource(userID, asset, end, amt); err != nil {
					return nil, nil, err
				}
			}
			if amt < 0 {
				owed.Amount = owed.Amount + uint64(-1*amt)
				// Apply owed interest (source is tracked by coupon period end date)
				if err = ubt.modifyUserInterestWithSource(userID, asset, end, amt); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	// Set position last updated time
	if err := ubt.setLastUpdated(userID, updateUnixTime); err != nil {
		return nil, nil, err
	}
	return earned, owed, nil
}

// UnlockCanceledOrderBalance calculates the remaining amountIn from an order and unlocks that balance from user's position.
func (ubt *UserPositionTracker) UnlockCanceledOrderBalance(order *types.Order) error {
	cancelled := order.State == types.OrderStateCanceled || order.State == types.OrderStatePartialExecuted
	if !cancelled {
		return nil
	}

	position, err := ubt.Position(order.UserUID)
	if err != nil {
		return err
	}

	balIn, err := order.BalanceIn()
	if err != nil {
		return err
	}
	inFilled, _, err := order.GetFilledAmts()
	if err != nil {
		return err
	}
	if balIn.Amt() > inFilled.Int64() {
		position.Locked = position.Locked.SubAmountToZero(balIn.Asset, balIn.Amt()-inFilled.Int64())
	}

	// Apply changes
	ubt.userPositions[order.UserUID] = position
	return nil
}

// ApplyCouponTrade updates a user's interest balance when assets with future coupon payments enter or leave balance.
func (ubt *UserPositionTracker) ApplyCouponTrade(
	assets AssetData,
	userID string,
	// Fees are generated for assets entering or leaving user's balance (including as a result of borrowing)
	balanceGained, balanceLost *ltypes.Balance,
	// Unix time of the trade, combined with asset data, is used to compute progress toward asset coupon payment
	tradeUnixTime int64,
) (owed *ltypes.Balance, earned *ltypes.Balance, err error) {
	owed = ltypes.ZeroBalance("Interest")
	earned = ltypes.ZeroBalance("Interest")
	interestAssetExponent, err := assets.Decimals("Interest")
	if err != nil {
		return nil, nil, err // could not get asset
	}
	if assets.HasCoupon(balanceGained.Asset) {
		gainAssetExponent, err := assets.Decimals(balanceGained.Asset)
		if err != nil {
			return nil, nil, err // could not get asset
		}
		// Get coupon amount and progress from asset
		gainAsset := balanceGained.Asset
		start, end, yield, err := assets.CurrentCouponPeriod(gainAsset, tradeUnixTime)
		if err != nil {
			return nil, nil, err // could not get period
		}
		couponPeriodLength := float64(end - start)
		couponPeriodElapsed := float64(tradeUnixTime - start)
		if couponPeriodLength > 0 && couponPeriodElapsed > 0 && yield > 0 {
			// some of the coupon period had elapsed before user bought this asset
			progress := couponPeriodElapsed / couponPeriodLength // 0 < progress < 1
			// User owes interest because they bought progress towards a coupon payment
			interestAmountBeforeExp := progress * yield * balanceGained.AmtFloat()
			// Apply asset exponents
			interestAmount := math.Pow10(interestAssetExponent-gainAssetExponent) * interestAmountBeforeExp
			amt := int64(interestAmount)
			if amt > 0 {
				owed = ltypes.NewBalance("Interest", amt)
				// Apply owed interest (this does NOT set position lastUpdateTime)
				if err = ubt.modifyUserInterestWithSource(userID, gainAsset, end, -1*amt); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	if assets.HasCoupon(balanceLost.Asset) {
		lostAssetExponent, err := assets.Decimals(balanceLost.Asset)
		if err != nil {
			return nil, nil, err // could not get asset
		}
		// Get coupon amount and progress from asset
		lostAsset := balanceLost.Asset
		start, end, yield, err := assets.CurrentCouponPeriod(lostAsset, tradeUnixTime)
		if err != nil {
			return nil, nil, err // could not get period
		}
		couponPeriodLength := float64(end - start)
		couponPeriodElapsed := float64(tradeUnixTime - start)
		if couponPeriodLength > 0 && couponPeriodElapsed > 0 && yield > 0 {
			// some of the coupon period had elapsed before user sold this asset
			progress := couponPeriodElapsed / couponPeriodLength // 0 < progress < 1
			// user earns interest because they sold progress towards a coupon payment
			interestAmountBeforeExp := progress * yield * balanceLost.AmtFloat()
			// Apply asset exponents
			interestAmount := math.Pow10(interestAssetExponent-lostAssetExponent) * interestAmountBeforeExp
			amt := int64(interestAmount)
			if amt > 0 {
				earned = ltypes.NewBalance("Interest", amt)
				// Apply earned interest (this does NOT set position lastUpdateTime)
				if err = ubt.modifyUserInterestWithSource(userID, lostAsset, end, amt); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	return owed, earned, nil
}

// ActivateStablecoinEquivalence fills a user's balance until it reaches a given amount of stablecoins, by
// removing other stablecoins. No-op if not stablecoin.
func (ubt *UserPositionTracker) ActivateStablecoinEquivalence(
	assets AssetData, userID string, balanceRequired *ltypes.Balance,
) {
	// Ensure balances are initialized
	position, err := ubt.Position(userID)
	if err != nil {
		return // could not get position
	}
	// Modify position
	position = activateStablecoinEquivalence(assets, position, balanceRequired)
	// Track position
	ubt.userPositions[userID] = position
}

// activateStablecoinEquivalence fills a user's balance until it reaches a given amount of stablecoins, by
// removing other stablecoins. No-op if not stablecoin.
func activateStablecoinEquivalence(
	assets AssetData, position *ltypes.Position, balanceRequired *ltypes.Balance,
) *ltypes.Position {
	if !assets.IsCurrency(balanceRequired.Asset) {
		return position // not a stablecoin
	}
	if position.Owned.HasAtLeast(balanceRequired.Asset, balanceRequired.Amt()) {
		return position // not necessary
	}
	decimals, err := assets.Decimals(balanceRequired.Asset)
	if err != nil {
		return position // could not get asset
	}
	// Convert stablecoins - potentially multiple types - to target asset
	dollarsToConvert := balanceRequired.Amt() - position.Owned.AmountOf(balanceRequired.Asset)
	// Amount to convert in whole dollars, rounding up. For example, $2.31 becomes 3
	dollarsToConvert = applyDecimalsThenRound(dollarsToConvert, decimals, true)
	dollarsConverted := int64(0)
	for _, assetID := range position.Owned.PositiveAssets() {
		if dollarsConverted >= dollarsToConvert {
			return position // success
		}
		if !assets.IsCurrency(assetID) || assetID == balanceRequired.Asset {
			continue // asset cannot be converted
		}
		cDecimals, err := assets.Decimals(assetID)
		if err != nil {
			continue // could not get asset
		}
		// Amount of this single asset we can convert, in whole dollars, rounding down. For example, $2.31 becomes 2
		amt := position.Owned.AmountOf(assetID)
		dollars := applyDecimalsThenRound(amt, cDecimals, false)
		if dollars > (dollarsToConvert - dollarsConverted) {
			dollars = dollarsToConvert - dollarsConverted // prevent overshooting goal
		}
		if dollars <= 0 {
			continue // less than $1 will not be converted
		}
		// Compute balance changes
		gain := ltypes.NewBalance(balanceRequired.Asset, dollars*exp10(decimals))
		loss := ltypes.NewBalance(assetID, dollars*exp10(cDecimals))
		if position.Owned, err = position.Owned.SafeSub(loss.Asset, loss.Amt()); err != nil {
			continue // error means balance not mutated. This is safe.
		}
		position.Owned = position.Owned.AddAmount(gain.Asset, gain.Amt())
		position.SSEQ = position.SSEQ.SubAmount(loss.Asset, loss.Amt())
		position.SSEQ = position.SSEQ.AddAmount(gain.Asset, gain.Amt())
		dollarsConverted += dollars
	}
	return position
}

// CleanupStablecoinEquivalence attempts to convert stablecoin equivalence positions back to their
// original balances if available.
func (ubt *UserPositionTracker) CleanupStablecoinEquivalence(assets AssetData, userID string) {
	// SSEQ will be tracked in user position
	position, err := ubt.Position(userID)
	if err != nil {
		return // could not get position
	}
	// Get a list of all balances user has lost due to simple stablecoin equivalence (usually their native asset)
	negativeSSEQ := position.SSEQ.Negative()
	// Get a list of all balances user has gained due to simple stablecoin equivalence
	positiveSSEQ := position.SSEQ.Positive()
	// Check for no-op
	if len(negativeSSEQ.AssetIDs()) < 1 || len(positiveSSEQ.AssetIDs()) < 1 {
		return // nothing to net
	}
	negAssetID := negativeSSEQ.AssetIDs()[0]
	posAssetID := positiveSSEQ.AssetIDs()[0]
	negBal := negativeSSEQ.SelectPositive(negAssetID)
	posBal := positiveSSEQ.SelectPositive(posAssetID)
	if !assets.IsCurrency(negAssetID) {
		return // not a stablecoin
	}
	if !assets.IsCurrency(posAssetID) {
		return // not a stablecoin
	}
	// determine the minimum exponent between the two assets - this determines our maximum precision
	posExponent, err := assets.Decimals(posAssetID)
	if err != nil {
		return
	}
	negExponent, err := assets.Decimals(negAssetID)
	if err != nil {
		return
	}
	minimumExponent := min(posExponent, negExponent)
	// Determine the amount of assets in balances which can be converted back to native asset.
	// For example, if positive asset exponent is 3 and negative asset exponent is 2, then:
	// - the minimum exponent is 2
	// - 4432 negative asset (44.32) remains 4432
	// - 69421 positive asset (69.421) becomes 6942.
	amtOwned := applyDecimalsThenRound(
		position.Owned.AmountOf(posAssetID),
		posExponent-minimumExponent,
		false,
	)
	amtPosEquivalence := applyDecimalsThenRound(
		posBal.Amt(),
		posExponent-minimumExponent,
		false,
	)
	amtNegEquivalence := applyDecimalsThenRound(
		negBal.Amt(),
		negExponent-minimumExponent,
		false,
	)
	// The minimum of the above amounts after conversion
	// In the above example, min(4432,6942) = 4432
	amountToConvertAtMinExponent := min(amtOwned, amtPosEquivalence, amtNegEquivalence)
	if amountToConvertAtMinExponent <= 0 {
		return // nothing can be done
	}
	// Compute balance changes
	gainAssetID := negBal.Asset
	lossAssetID := posBal.Asset
	// For example, if positive asset exponent was 3 and negative asset exponent was 2,
	// then the amount of 4432 from above must become 44320 positive asset and 4432 negative asset
	gainAmount := amountToConvertAtMinExponent * exp10(negExponent-minimumExponent)
	lossAmount := amountToConvertAtMinExponent * exp10(posExponent-minimumExponent)
	if position.Owned, err = position.Owned.SafeSub(lossAssetID, lossAmount); err != nil {
		return
	}
	position.Owned = position.Owned.AddAmount(gainAssetID, gainAmount)
	position.SSEQ = position.SSEQ.SubAmount(lossAssetID, lossAmount)
	position.SSEQ = position.SSEQ.AddAmount(gainAssetID, gainAmount)
	// Apply changes
	ubt.userPositions[userID] = position
}
