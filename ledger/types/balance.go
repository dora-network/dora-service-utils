package types

import (
	"errors"
	"fmt"
	"time"

	"github.com/goccy/go-json"
)

// Balance contains an asset ID and a uint64 amount.
// Zero-valued Balance is invalid due to empty asset ID.
type Balance struct {
	Asset  string `json:"asset" redis:"asset"`
	Amount uint64 `json:"amount" redis:"amount"`
}

type Integer64 interface {
	int64 | uint64
}

func (b *Balance) MarshalBinary() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Balance) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b)
}

// NewBalance creates a Balance.
// If asset ID is empty, then all fields are zero-valued and Valid() will return false.
func NewBalance[T Integer64](asset string, amount T) *Balance {
	amt := uint64(0)
	if amount > 0 {
		amt = uint64(amount)
	}
	if asset != "" {
		return &Balance{
			Asset:  asset,
			Amount: amt,
		}
	}
	return &Balance{}
}

// ZeroBalance creates a Balance with zero amount.
func ZeroBalance(asset string) *Balance {
	if asset != "" {
		return &Balance{
			Asset:  asset,
			Amount: 0,
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

// Sub from a balance. Error on negative result, mismatched assets, or invalid assets.
func (b *Balance) Sub(sub *Balance) (*Balance, error) {
	if sub == nil {
		return b, nil // no-op is safe
	}
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
	var err error
	result := b.Copy()
	result, err = result.Sub(NewBalance(b.Asset, amt))
	return result, err == nil
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
