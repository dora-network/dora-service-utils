package ledger

import (
	"fmt"
	"github.com/dora-network/dora-service-utils/errors"
	"github.com/dora-network/dora-service-utils/math"
)

type Amount struct {
	AssetID string `json:"asset_id"`
	Amount  uint64 `json:"amount"`
}

func ZeroAmount(assetID string) Amount {
	return NewAmount(assetID, 0)
}

func NewAmount(assetID string, amount uint64) Amount {
	return Amount{
		AssetID: assetID,
		Amount:  amount,
	}
}

// Equal returns true if one Amount is equal to another.
func (a Amount) Equal(x Amount) bool {
	return a.AssetID == x.AssetID && a.Amount == x.Amount
}

// Copy returns a copied Amount.
func (a Amount) Copy() Amount {
	return NewAmount(a.AssetID, a.Amount)
}

// Match returns true if two Amounts have the same AssetID
func (a Amount) Match(bal Amount) bool {
	return a.AssetID == bal.AssetID
}

// Add an Amount to this one, returning the result and an error if asset IDs do not match.
func (a Amount) Add(amt Amount) (Amount, error) {
	if !a.Match(amt) {
		return Amount{}, errors.New(errors.InternalError, "Amount.Add: AssetIDs did not match")
	}
	result, err := math.CheckedAddU64(a.Amount, amt.Amount)
	if err != nil {
		return Amount{}, err
	}
	return Amount{
		AssetID: a.AssetID,
		Amount:  result,
	}, nil
}

// Sub an Amount from this one, returning the result and an error if asset IDs do not match.
// Also returns an error if the final amount would be negative.
func (a Amount) Sub(amt Amount) (Amount, error) {
	if !a.Match(amt) {
		return Amount{}, errors.New(errors.InternalError, "Amount.Sub: AssetIDs did not match")
	}
	result, err := math.CheckedSubU64(a.Amount, amt.Amount)
	if err != nil {
		return Amount{}, err
	}
	return Amount{
		AssetID: a.AssetID,
		Amount:  result,
	}, nil
}

// SubToZero an Amount from this one, returning the result, the subbed amount and an error if asset IDs do not match.
// If the final amount would be negative returns zero and the real subbed amount.
// If not the subbed amount is equal to amt.
func (a Amount) SubToZero(amt Amount) (Amount, Amount, error) {
	if !a.Match(amt) {
		return Amount{}, Amount{}, errors.New(errors.InternalError, "Amount.Sub: AssetIDs did not match")
	}
	if a.LT(amt) {
		amt = a.Copy()
	}
	result, _ := math.CheckedSubU64(a.Amount, amt.Amount)
	return Amount{AssetID: a.AssetID, Amount: result},
		amt,
		nil
}

// LT returns true if an Amount is less than another.
func (a Amount) LT(x Amount) bool {
	return a.Amount < x.Amount
}

// LTE returns true if an Amount is less or equal than another.
func (a Amount) LTE(x Amount) bool {
	return a.Amount <= x.Amount
}

// GT returns true if an Amount is greater than another.
func (a Amount) GT(x Amount) bool {
	return a.Amount > x.Amount
}

// GTE returns true if an Amount is greater or equal than another.
func (a Amount) GTE(x Amount) bool {
	return a.Amount >= x.Amount
}

// Validate requires a non-empty assetID.
func (a Amount) Validate() error {
	if a.AssetID == "" {
		return errors.Data("Amount with empty AssetID")
	}
	return nil
}

// IsZero returns true if a Amount is zero.
func (a Amount) IsZero() bool {
	return a.Amount == 0
}

func (a Amount) String() string {
	return fmt.Sprintf("%d %s", a.Amount, a.AssetID)
}
