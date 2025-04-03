package helpers

import ltypes "github.com/dora-network/dora-service-utils/ledger/types"

// ActivateStablecoinEquivalence fills a user's balance until it reaches a given amount of stablecoins, by
// removing other stablecoins. No-op if not stablecoin.
func ActivateStablecoinEquivalence(
	assets AssetData,
	position *ltypes.Position,
	balanceRequired *ltypes.Balance,
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
	return position.Copy()
}

// CleanupStablecoinEquivalence attempts to convert stablecoin equivalence positions back to their
// original balances if available.
func CleanupStablecoinEquivalence(assets AssetData, position *ltypes.Position) *ltypes.Position {
	// Get a list of all balances user has lost due to simple stablecoin equivalence (usually their native asset)
	negativeSSEQ := position.SSEQ.Negative()
	// Get a list of all balances user has gained due to simple stablecoin equivalence
	positiveSSEQ := position.SSEQ.Positive()
	// Check for no-op
	if len(negativeSSEQ.AssetIDs()) < 1 || len(positiveSSEQ.AssetIDs()) < 1 {
		return position // nothing to net
	}
	negAssetID := negativeSSEQ.AssetIDs()[0]
	posAssetID := positiveSSEQ.AssetIDs()[0]
	negBal := negativeSSEQ.SelectPositive(negAssetID)
	posBal := positiveSSEQ.SelectPositive(posAssetID)
	if !assets.IsCurrency(negAssetID) {
		return position // not a stablecoin
	}
	if !assets.IsCurrency(posAssetID) {
		return position // not a stablecoin
	}
	// determine the minimum exponent between the two assets - this determines our maximum precision
	posExponent, err := assets.Decimals(posAssetID)
	if err != nil {
		return position
	}
	negExponent, err := assets.Decimals(negAssetID)
	if err != nil {
		return position
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
		return position // nothing can be done
	}
	// Compute balance changes
	gainAssetID := negBal.Asset
	lossAssetID := posBal.Asset
	// For example, if positive asset exponent was 3 and negative asset exponent was 2,
	// then the amount of 4432 from above must become 44320 positive asset and 4432 negative asset
	gainAmount := amountToConvertAtMinExponent * exp10(negExponent-minimumExponent)
	lossAmount := amountToConvertAtMinExponent * exp10(posExponent-minimumExponent)
	if position.Owned, err = position.Owned.SafeSub(lossAssetID, lossAmount); err != nil {
		return position
	}
	position.Owned = position.Owned.AddAmount(gainAssetID, gainAmount)
	position.SSEQ = position.SSEQ.SubAmount(lossAssetID, lossAmount)
	position.SSEQ = position.SSEQ.AddAmount(gainAssetID, gainAmount)

	return position.Copy()
}

// applyDecimalsThenRound divides n by 10^x then optionally rounds up
func applyDecimalsThenRound(n int64, x int, roundUp bool) int64 {
	m := exp10(x)
	result := n / m
	if result*m < n && roundUp {
		return result + 1 // rounded up
	}
	return result // n / 10^x was an exact integer, or roundUp was false
}

// exp10 returns 10^x for positive x; 1 otherwise
func exp10(x int) int64 {
	result := int64(1)
	for i := 0; i < x; i++ {
		result *= 10
	}
	return result
}
