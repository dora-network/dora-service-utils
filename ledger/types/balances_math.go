package types

import "fmt"

// AmountOf a given asset in Balances. Result can be negative.
func (b *Balances) AmountOf(assetID string) int64 {
	if b.Bals == nil {
		return 0
	}
	return b.Bals[assetID]
}

// AddAmount a given amount of a single asset to Balances and return the result.
// Original Balances object is not mutated. Result can be negative.
// Negative input is equivalent to using Balances.Sub instead.
// No effect on empty asset ID or zero amount.
func (b *Balances) AddAmount(assetID string, amount int64) *Balances {
	result := b.Copy() // copy to prevent mutation of original
	if assetID != "" {
		result.Bals[assetID] = result.Bals[assetID] + amount
	}
	return result
}

// Add one or more balances to Balances.
// Original Balances object is not mutated. Result can be negative.
// Negative input is equivalent to using Balances.Sub instead.
// Inputs with empty asset ID or zero amount are ignored.
func (b *Balances) Add(adds ...*Balance) *Balances {
	result := b.Copy()
	for _, add := range adds {
		result = result.AddAmount(add.Asset, add.Amt())
	}
	return result
}

// AddBals to Balances and return the result.
// Original Balances object is not mutated. Result can be negative.
func (b *Balances) AddBals(add *Balances) *Balances {
	result := b.Copy() // copy to prevent mutation of original
	for id, amt := range add.Bals {
		result.Bals[id] = result.Bals[id] + amt
	}
	return result
}

// Invert Balances. Positive amounts become negative, and vice versa.
// Original Balances object is not mutated.
func (b *Balances) Invert() *Balances {
	result := b.Copy() // copy to prevent mutation of original
	for id, amt := range result.Bals {
		result.Bals[id] = -1 * amt
	}
	return result
}

// SubAmount a given amount of a single asset from Balances and return the result.
// Original Balances object is not mutated. Result can be negative.
// Negative input is equivalent to using Balances.Add instead
func (b *Balances) SubAmount(assetID string, amount int64) *Balances {
	return b.AddAmount(assetID, amount*-1)
}

// Sub one or more balances from Balances.
// Original Balances object is not mutated. Result can be negative.
// Negative input is equivalent to using Balances.Add instead.
// Inputs with empty asset ID or zero amount are ignored.
func (b *Balances) Sub(subs ...*Balance) *Balances {
	result := b.Copy()
	for _, sub := range subs {
		result = result.SubAmount(sub.Asset, sub.Amt())
	}
	return result
}

// SafeSub a given amount of a single asset from Balances and return the result.
// Original Balances object is not mutated. Error if result would be negative or input is negative.
func (b *Balances) SafeSub(assetID string, amount int64) (*Balances, error) {
	if amount < 0 {
		return b, fmt.Errorf("cannot sub: %d %s is negative", amount, assetID)
	}
	if b.Bals == nil {
		return b, fmt.Errorf("cannot sub: %d %s from nil", amount, assetID)
	}
	if !b.HasAtLeast(assetID, amount) {
		return b, fmt.Errorf("cannot sub: %d - %d (%s)", b.Bals[assetID], amount, assetID)
	}
	return b.SubAmount(assetID, amount), nil
}

// SubToZero Balances from Balances and return the result.
// Any surplus value that could not be subtracted is returned separately.
// Negative or zero input is a no-op.
func (b *Balances) SubToZero(subs *Balances) (result, surplus *Balances) {
	result = b.Copy()
	surplus = EmptyBalances()
	for _, id := range subs.AssetIDs() {
		sub := NewBalance(id, uint64(subs.AmountOf(id)))
		var s *Balance
		result, s = result.SubBalToZero(sub)
		surplus = surplus.Add(s)
	}
	return
}

// SubBalToZero a single Balance from Balances and return the result.
// Any surplus value that could not be subtracted is returned separately.
// Negative or zero input is a no-op.
func (b *Balances) SubBalToZero(sub *Balance) (result *Balances, surplus *Balance) {
	result = b.Copy()
	if b.Bals == nil || b.AmountOf(sub.Asset) <= 0 {
		return result, sub // nothing to do (preserves existing negative or zero amounts)
	}
	naiveResultAmt := b.AmountOf(sub.Asset) - sub.Amt()
	if naiveResultAmt < 0 {
		// naive sub would result in negative amount:
		// sub to exactly zero, and return surplus amount
		return b.SubAmount(sub.Asset, b.AmountOf(sub.Asset)), NewBalance(sub.Asset, uint64(naiveResultAmt*-1))
	}
	// sub to a zero or positive amount. surplus is zero.
	return b.SubAmount(sub.Asset, sub.Amt()), NewBalance(sub.Asset, int64(0))
}

// SubAmountToZero a given amount of a single asset from Balances and return the result.
// Negative or zero input is a no-op.
func (b *Balances) SubAmountToZero(assetID string, amount int64) *Balances {
	if b.Bals == nil || b.AmountOf(assetID) <= 0 || amount <= 0 {
		return b // nothing to do (preserves existing negative or zero amounts)
	}
	if b.AmountOf(assetID) <= amount {
		return b.SubAmount(assetID, b.Bals[assetID]) // sub to exactly zero
	}
	return b.SubAmount(assetID, amount) // sub to a nonzero amount
}

// SubBals from Balances and return the result.
// Original Balances object is not mutated. Result can be negative.
func (b *Balances) SubBals(sub *Balances) *Balances {
	return b.AddBals(sub.Invert())
}

// HasAtLeast returns true if a balance has at least the given amount.
// For negative input, returns true if the balance's amount of assetID is more negative than the input amount.
func (b *Balances) HasAtLeast(assetID string, amount int64) bool {
	if amount == 0 {
		return true
	}
	if amount < 0 {
		return b.Bals != nil && b.Bals[assetID] <= amount // negatve case
	}
	return b.Bals != nil && b.Bals[assetID] >= amount // usual case
}

// HasNegative returns true if balances have at least one negative value
func (b *Balances) HasNegative() bool {
	if b.Bals != nil {
		for _, amt := range b.Bals {
			if amt < 0 {
				return true
			}
		}
	}
	return false
}
