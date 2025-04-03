package types

import (
	"github.com/goccy/go-json"

	"github.com/dora-network/dora-service-utils/errors"
)

// Position contains a snapshot of all of a user's assets and debts.
type Position struct {
	// UserID identifies the owner of the position
	UserID string `json:"user_id" redis:"user_id"`

	// Assets owned (including bonds and currencies). Negative values indicate borrows.
	Owned *Balances `json:"owned" redis:"owned"`
	// Assets locked as potential inputs to user open orders. Subset of positive Owned amounts.
	Locked *Balances `json:"locked" redis:"locked"`
	// Assets supplied to module but not yet withdrawn.
	Supplied *Balances `json:"supplied" redis:"supplied"`
	// Effects of simple stablecoin equivalence on user balance.
	// Positive values indicate assets gained, and negative values indicate assets lost.
	// Assets are lost and gained 1:1, so the sum of positive and negative amounts after decimals will always be 0.
	SSEQ *Balances `json:"sseq" redis:"sseq"`
	// Assets which have been withheld from a user's Owned balance for technical reasons.
	Inactive *Balances `json:"inactive" redis:"inactive"`
	// InterestSources contains supplemental information about Owned "Interest" balances related to coupon payments.
	// We recycle the Balances struct here, containing map[string]int64, but rather than representing a set of
	// asset balances like map[AssetID]Amount, each entry in this map represents the amount of AssetID="Interest"
	// present in Position.Owned which came from a particular bond's specific coupon period.
	// For example, if InterestSources["Bond_A-Coupon_123456"]=789, then $7.89 of this user's owned interest
	// came from asset Bond_A's coupon period ending at unix timestamp 123456. Note that the single hyphen
	// makes the key pass asset ID validation rules, even though it is not any asset's AssetID.
	// Also note that a negative value (for example, -789) is valid here and would refer to the user owing
	// interest due to having bought a bond mid-coupon-period.
	InterestSources *Balances `json:"interest_sources" redis:"interest_sources"`

	// Native stablecoin asset which the user originally deposited and will prefer for withdrawals
	NativeAsset string `json:"native_asset" redis:"native_asset"`

	// Unix time when position was last updated. Should only be set when position is modified by a transaction.
	LastUpdated int64 `json:"last_updated" redis:"last_updated"`
	// Sequence number of the position. A user's first position on the platform has sequence number 1,
	// and each time their position is modified by a transaction, it increments. Ensures completeness or records.
	Sequence uint64 `json:"sequence" redis:"sequence"`

	// Internal usage only - supports isModified by remembering the entire position's original string representation
	original         string
	originalSequence uint64
}

func (p *Position) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Position) UnmarshalBinary(data []byte) error {
	err := json.Unmarshal(data, p)
	p.Init() // Init must be called after json unmarshaling
	return err
}

// Init sets a position's original field to its current json representation. No-op if already set.
// For a position object to be valid, this must be called after json unmarshaling.
func (p *Position) Init() {
	// Nil balances should not be allowed
	if p.Owned == nil {
		p.Owned = EmptyBalances()
	}
	if p.Locked == nil {
		p.Locked = EmptyBalances()
	}
	if p.Supplied == nil {
		p.Supplied = EmptyBalances()
	}
	if p.SSEQ == nil {
		p.SSEQ = EmptyBalances()
	}
	if p.Inactive == nil {
		p.Inactive = EmptyBalances()
	}
	if p.InterestSources == nil {
		p.InterestSources = EmptyBalances()
	}
	// Store original state. No-op if already stored.
	if p.original == "" {
		j, err := json.Marshal(p)
		if err != nil {
			return
		}
		// Track exported fields for IsModified
		p.original = string(j)
		p.originalSequence = p.Sequence
	}
}

// InitialPosition returns a position with zero balances and a sequence number of 0,
// representing an account's state before its first activity.
func InitialPosition(userID string) *Position {
	p := &Position{
		UserID: userID,
		// Tracking fields
		NativeAsset: "",
		Sequence:    0,
		LastUpdated: 0,
		// Note: All *Balances fields are set to EmptyBalances by p.Init()
	}
	// Track exported fields for IsModified, as well as initial Sequence
	p.Init()
	return p
}

func NewPosition(
	userID string,
	owned, locked, supplied, sseq, inactive, interestSources *Balances,
	nativeAsset string,
	lastUpdated int64,
	sequence uint64,
) (*Position, error) {
	p := &Position{
		UserID: userID,
		// Balances (can Validate)
		Owned:           owned,
		Locked:          locked,
		Supplied:        supplied,
		SSEQ:            sseq,
		Inactive:        inactive,
		InterestSources: interestSources,
		// Tracking fields
		NativeAsset: nativeAsset,
		Sequence:    sequence,
		LastUpdated: lastUpdated,
	}
	// Track exported fields for IsModified, as well as initial Sequence
	// Also overrides any nil *Balances with EmptyBalances()
	p.Init()
	return p, p.Validate()
}

// Validate that Position does not contain any invalid Balances or empty required fields
func (p *Position) Validate() error {
	if p.UserID == "" {
		return errors.Data("empty user ID in Position")
	}
	// Balances.Validate will panic if Balances are nil, so we check first
	if p.Owned == nil ||
		p.Locked == nil ||
		p.Supplied == nil ||
		p.SSEQ == nil ||
		p.Inactive == nil ||
		p.InterestSources == nil {
		return errors.Data("nil Balances in Position")
	}
	if err := p.Owned.Validate(true); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "position Owned")
	}
	if err := p.Locked.Validate(false); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "position Locked")
	}
	if err := p.Supplied.Validate(false); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "position Supplied")
	}
	if err := p.SSEQ.Validate(true); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "position SSEQ")
	}
	if err := p.InterestSources.Validate(true); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "position Interest Sources")
	}
	if err := p.Inactive.Validate(true); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "position Inactive")
	}
	if err := ValidAssetID(p.NativeAsset); err != nil && p.NativeAsset != "" {
		return err // notice that error is ignored if NativeAsset is not set
	}
	if p.original == "" {
		return errors.NewInternal("position not initialized")
	}
	return nil
}

// String representation of position. Always has updated sequence if modified. Does not mutate original.
func (p *Position) String() string {
	p.UpdateSequence() // no-op if unmodified or sequence already updated.
	return p.jsonString()
}

// internal usage only. marshal as string without trying to update sequence if IsModified.
func (p *Position) jsonString() string {
	j, _ := json.Marshal(p)
	return string(j)
}

// Original positon, represented as a string
func (p *Position) Original() string {
	return p.original
}

// Copy entire position, including original and isModified data.
func (p *Position) Copy() *Position {
	j, _ := json.Marshal(p)
	position := &Position{
		// Any unexported fields must be copied here, before unmarshal
		original:         p.original,
		originalSequence: p.originalSequence,
	}
	_ = json.Unmarshal(j, position)
	return position
}

// Snapshot entire position, setting current state as original and isModified to false.
func (p *Position) Snapshot() *Position {
	j, _ := json.Marshal(p)
	position := Position{}
	_ = json.Unmarshal(j, &position)
	position.Init()
	return &position
}

// IsModified detects whether a position's exported fields have changed at all since initialization.
func (p *Position) IsModified() bool {
	return p.original != p.jsonString()
}

// NextSequence returns position.originalSequence + 1
func (p *Position) NextSequence() uint64 {
	return p.originalSequence + 1
}

// UpdateSequence position.Sequence if position has been modified. no-op if unmodified or sequence already updated.
func (p *Position) UpdateSequence() {
	if p.IsModified() {
		p.Sequence = p.NextSequence()
	}
}

// ActivateStablecoinEquivalence fills a user's balance until it reaches a given amount of stablecoins, by
// removing other stablecoins. No-op if not stablecoin.
func (p *Position) ActivateStablecoinEquivalence(
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
