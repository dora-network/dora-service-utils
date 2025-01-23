package types

import (
	"errors"
	"fmt"
	"github.com/goccy/go-json"
	"time"
)

// Balance contains an asset ID and a uint64 amount.
// Zero-valued Balance is invalid due to empty asset ID.
type Balance struct {
	Asset  string `json:"asset" redis:"asset"`
	Amount uint64 `json:"amount" redis:"amount"`
}

func (b *Balance) MarshalBinary() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Balance) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b)
}

// NewBal creates a Balance.
// If asset ID is empty or amount is negative then all fields are zero-valued and Valid() will return false.
func NewBal(asset string, amount int64) *Balance {
	if asset != "" && amount >= 0 {
		return &Balance{
			Asset:  asset,
			Amount: uint64(amount),
		}
	}
	return &Balance{}
}

// NewBalance creates a Balance.
// If asset ID is empty, then all fields are zero-valued and Valid() will return false.
func NewBalance(asset string, amount uint64) *Balance {
	if asset != "" {
		return &Balance{
			Asset:  asset,
			Amount: amount,
		}
	}
	return &Balance{}
}

// Valid will only be true if Balance was constructed properly
func (b *Balance) Valid() bool {
	return b.Asset != ""
}

func (b *Balance) String() string {
	return fmt.Sprintf("%d %s", b.Amount, b.Asset)
}

func (b *Balance) Copy() *Balance {
	return NewBalance(b.Asset, b.Amount)
}

func (b *Balance) Amt() int64 {
	return int64(b.Amount)
}

func (b *Balance) AmtFloat() float64 {
	return float64(b.Amount)
}

func (b *Balance) IsZero() bool {
	return b.Amount == 0
}

func (b *Balance) IsPositive() bool {
	return b.Amount > 0
}

// Sub from a balance. Error on negative result, mismatched assets, or invalid assets.
func (b *Balance) Sub(sub *Balance) (*Balance, error) {
	if !b.Valid() || !sub.Valid() {
		return nil, errors.New("balance.Sub: invalid input")
	}
	if sub.Amount == 0 {
		return b, nil // no-op is safe
	}
	if b.Asset != sub.Asset {
		return nil, fmt.Errorf("balance.Sub: mismatched assets %s and %s", b.Asset, sub.Asset)
	}
	if b.Amount < sub.Amount {
		return nil, fmt.Errorf("balance.Sub (%s): result would be negative", b.Asset)
	}
	return NewBalance(b.Asset, b.Amount-sub.Amount), nil
}

// SubAmt from a balance. Returns ok if balance was valid and result was not negative.
func (b *Balance) SubAmt(amt uint64) (*Balance, bool) {
	if !b.Valid() || b.Amount < amt {
		return nil, false
	}
	return NewBalance(b.Asset, b.Amount-amt), true
}

// MulF a balance's amount by a float, rounding down. Error if f is negative.
func (b *Balance) MulF(f float64) (*Balance, error) {
	if f < 0 {
		return nil, errors.New("MulF by negative value")
	}
	amt := float64(b.Amount) * f
	return NewBalance(b.Asset, uint64(amt)), nil
}

type Interest struct {
	Earned      uint64    `json:"earned" redis:"earned"`
	Owed        uint64    `json:"owed" redis:"owed"`
	Claimed     uint64    `json:"claimed" redis:"claimed"`
	Paid        uint64    `json:"paid" redis:"paid"`
	LastUpdated time.Time `json:"last_updated" redis:"last_updated"`
}

func (i *Interest) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

func (i *Interest) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, i)
}
