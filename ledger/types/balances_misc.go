package types

import (
	"encoding/json"
	"github.com/dora-network/dora-service-utils/errors"
	"regexp"
	"strings"
)

// Validate that Balances does not contain any empty asset IDs
func (b *Balances) Validate(allowNegative bool) error {
	for assetID, amount := range b.Bals {
		if assetID == "" {
			return errors.Data("empty asset ID in Balances")
		}
		if err := ValidAssetID(assetID); err != nil {
			return err
		}
		if !allowNegative && amount < 0 {
			return errors.Data("negative amount in Balances: %d %s", amount, assetID)
		}
	}
	return nil
}

// SelectPositive a single Balance from Balances, if positive.
func (b *Balances) SelectPositive(assetID string) *Balance {
	result := NewBal(assetID, 0)
	if b.Bals != nil {
		amt := b.Bals[assetID]
		if amt > 0 {
			result.Amount = uint64(amt)
		}
	}
	return result
}

// Copy creates a safe copy of Balances.
func (b *Balances) Copy() *Balances {
	result := &Balances{
		Bals: map[string]int64{},
	}
	for k, v := range b.Bals {
		result.Bals[k] = v
	}
	return result
}

// Positive creates a safe copy of only positive Balances and returns them.
func (b *Balances) Positive() *Balances {
	result := &Balances{
		Bals: map[string]int64{},
	}
	for k, v := range b.Bals {
		if v > 0 {
			result.Bals[k] = v
		}
	}
	return result
}

// Negative creates a safe copy of only negative Balances, and returns them as positive amounts.
func (b *Balances) Negative() *Balances {
	result := &Balances{
		Bals: map[string]int64{},
	}
	for k, v := range b.Bals {
		if v < 0 {
			result.Bals[k] = -v
		}
	}
	return result
}

// Zeros creates a safe copy of zero Balances for requested ids and returns them.
func (b *Balances) Zeros(ids ...string) *Balances {
	result := &Balances{
		Bals: map[string]int64{},
	}

	for _, id := range ids {
		amt := b.AmountOf(id)
		if amt == 0 {
			result.Bals[id] = amt
		}
	}
	return result
}

// AssetIDs returns the asset IDs of all nonzero balances
func (b *Balances) AssetIDs() []string {
	result := []string{}
	for id, amt := range b.Bals {
		if amt != 0 {
			result = append(result, id)
		}
	}
	return result
}

// PositiveAssets returns the asset IDs of all positive balances
func (b *Balances) PositiveAssets() []string {
	result := []string{}
	for id, amt := range b.Bals {
		if amt > 0 {
			result = append(result, id)
		}
	}
	return result
}

// NegativeAssets returns the asset IDs of all negative balances
func (b *Balances) NegativeAssets() []string {
	result := []string{}
	for id, amt := range b.Bals {
		if amt < 0 {
			result = append(result, id)
		}
	}
	return result
}

// String
func (b *Balances) String() string {
	j, err := json.Marshal(b)
	if err != nil {
		return "{\"Error\": " + err.Error() + "}" // so data output is always json
	}
	return string(j)
}

// ValidAssetID checks that an asset ID contains only alphanumeric characters and underscores,
// as well as at most one hyphen somewhere in the middle, and is non-empty.
func ValidAssetID(id string) error {
	re := regexp.MustCompile("[^A-Za-z0-9_-]")
	trimmed := re.ReplaceAllLiteralString(id, "")
	if id == "" ||
		id != trimmed || // this checks whether the regexp removed any characters outside the accepted set
		strings.HasPrefix(id, "-") ||
		strings.HasSuffix(id, "-") ||
		strings.Count(id, "-") > 1 {
		return errors.Data("invalid asset ID: %s", id)
	}
	return nil
}
