package types

import (
	"fmt"

	"github.com/goccy/go-json"

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

func (a *Amount) MarshalBinary() ([]byte, error) {
	return json.Marshal(a)
}

func (a *Amount) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, a)
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

// AddUint64 a uint64 amount to this Amount, returning the resulting Amount and an error if an overflow occurs.
// The assetID must be checked before calling this function.
func (a Amount) AddUint64(amt uint64) (Amount, error) {
	result, err := math.CheckedAddU64(a.Amount, amt)
	if err != nil {
		return Amount{}, fmt.Errorf("Amount.Add: %w", err)
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

// SubUint64 a uint64 amount from this Amount, returning the resulting Amount and an error if an underflow occurs.
// The assetID must be checked before calling this function.
func (a Amount) SubUint64(amt uint64) (Amount, error) {
	result, err := math.CheckedSubU64(a.Amount, amt)
	if err != nil {
		return Amount{}, fmt.Errorf("Amount.Sub: %w", err)
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
	if a.LTUint64(amt.Amount) {
		amt = a.Copy()
	}
	result, _ := math.CheckedSubU64(a.Amount, amt.Amount)
	return Amount{AssetID: a.AssetID, Amount: result},
		amt,
		nil
}

// SubToZeroUint64 subtracts a uint64 amount from this Amount, returning the resulting Amount and the subtracted amount.
// If the result would be negative, it returns zero and the original amount instead of throwing an error.
// The assetID must be checked before calling this function.
func (a Amount) SubToZeroUint64(amt uint64) (Amount, uint64, error) {
	if a.Amount < amt {
		return Amount{AssetID: a.AssetID, Amount: 0}, a.Amount, nil
	}
	result, _ := math.CheckedSubU64(a.Amount, amt)
	return Amount{AssetID: a.AssetID, Amount: result}, amt, nil
}

// LT returns true if an Amount is less than another.
func (a Amount) LT(x Amount) (bool, error) {
	if !a.Match(x) {
		return false, errors.New(errors.InternalError, "Amount.Sub: AssetIDs did not match")
	}
	return a.Amount < x.Amount, nil
}

// LTUint64 returns true if this Amount is less than a uint64 value.
func (a Amount) LTUint64(amt uint64) bool {
	return a.Amount < amt
}

// LTE returns true if an Amount is less or equal than another.
func (a Amount) LTE(x Amount) (bool, error) {
	if !a.Match(x) {
		return false, errors.New(errors.InternalError, "Amount.Sub: AssetIDs did not match")
	}
	return a.Amount <= x.Amount, nil
}

// LTEUint64 returns true if this Amount is less than or equal to a uint64 value.
func (a Amount) LTEUint64(amt uint64) bool {
	return a.Amount <= amt
}

// GT returns true if an Amount is greater than another.
func (a Amount) GT(x Amount) (bool, error) {
	if !a.Match(x) {
		return false, errors.New(errors.InternalError, "Amount.Sub: AssetIDs did not match")
	}
	return a.Amount > x.Amount, nil
}

// GTUint64 returns true if this Amount is greater than a uint64 value.
func (a Amount) GTUint64(amt uint64) bool {
	return a.Amount > amt
}

// GTE returns true if an Amount is greater or equal than another.
func (a Amount) GTE(x Amount) (bool, error) {
	if !a.Match(x) {
		return false, errors.New(errors.InternalError, "Amount.Sub: AssetIDs did not match")
	}
	return a.Amount >= x.Amount, nil
}

// GTEUint64 returns true if this Amount is greater than or equal to a uint64 value.
func (a Amount) GTEUint64(amt uint64) bool {
	return a.Amount >= amt
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
