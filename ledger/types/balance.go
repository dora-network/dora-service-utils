package types

import (
	"fmt"

	"github.com/goccy/go-json"

	"github.com/dora-network/dora-service-utils/errors"
	"github.com/dora-network/dora-service-utils/math"
)

type Balance struct {
	UserID  string `json:"user_id" redis:"user_id"`
	AssetID string `json:"asset_id" redis:"asset_id"`
	// Available Balance
	Balance    uint64 `json:"balance" redis:"balance"`
	Borrowed   uint64 `json:"borrowed" redis:"borrowed"`
	Collateral uint64 `json:"collateral" redis:"collateral"`
	Supplied   uint64 `json:"supplied" redis:"supplied"`
	Virtual    uint64 `json:"virtual" redis:"virtual"`
	Locked     uint64 `json:"locked" redis:"locked"`
}

func (b *Balance) MarshalBinary() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Balance) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b)
}

// IsZero returns true if all the Amount of the balance are zero.
func (b *Balance) IsZero() bool {
	if b == nil {
		return true
	}
	return b.Balance == 0 &&
		b.Borrowed == 0 &&
		b.Collateral == 0 &&
		b.Supplied == 0 &&
		b.Virtual == 0 &&
		b.Locked == 0
}

// Equal returns true if one Balance is equal to another.
func (b *Balance) Equal(x *Balance) bool {
	if b == nil || x == nil {
		return false
	}
	if b.UserID != x.UserID {
		return false
	}
	if b.AssetID != x.AssetID {
		return false
	}

	return b.Balance == x.Balance &&
		b.Borrowed == x.Borrowed &&
		b.Collateral == x.Collateral &&
		b.Supplied == x.Supplied &&
		b.Virtual == x.Virtual &&
		b.Locked == x.Locked
}

func NewBalance(userID, assetID string, balance, borrowed, collateral, supplied, virtual, locked uint64) *Balance {
	return &Balance{
		UserID:     userID,
		AssetID:    assetID,
		Balance:    balance,
		Borrowed:   borrowed,
		Collateral: collateral,
		Supplied:   supplied,
		Virtual:    virtual,
		Locked:     locked,
	}
}

func ZeroBalance(userID, assetID string) *Balance {
	return &Balance{
		UserID:     userID,
		AssetID:    assetID,
		Balance:    0,
		Borrowed:   0,
		Collateral: 0,
		Supplied:   0,
		Virtual:    0,
		Locked:     0,
	}
}

// Copy returns a copy of a Balance
func (b *Balance) Copy() *Balance {
	return &Balance{
		UserID:     b.UserID,
		AssetID:    b.AssetID,
		Balance:    b.Balance,
		Borrowed:   b.Borrowed,
		Collateral: b.Collateral,
		Supplied:   b.Supplied,
		Virtual:    b.Virtual,
		Locked:     b.Locked,
	}
}

// Match returns true if Balance and Amount have the same AssetID
func (b *Balance) Match(a Amount) bool {
	return a.AssetID == b.AssetID
}

// Add an Amount to Balance.Balance.
func (b *Balance) Add(amount Amount) error {
	if !b.Match(amount) {
		return errors.New(errors.InternalError, "Balance.Add: AssetIDs did not match")
	}

	result, err := math.CheckedAddU64(b.Balance, amount.Amount)
	if err != nil {
		return err
	}
	b.Balance = result
	return nil
}

// Sub an Amount from Balance.Balance.
func (b *Balance) Sub(amount Amount) error {
	if !b.Match(amount) {
		return errors.New(errors.InternalError, "Balance.Add: AssetIDs did not match")
	}

	result, err := math.CheckedSubU64(b.Balance, amount.Amount)
	if err != nil {
		return err
	}
	b.Balance = result
	return nil
}

// Lock adds an Amount to Balance.Locked. Returns an error if the result Balance.Locked
// is greater than the whole Balance.Balance
func (b *Balance) Lock(amount Amount) error {
	balance, err := math.CheckedSubU64(b.Balance, amount.Amount)
	if err != nil {
		return err
	}
	locked, err := math.CheckedAddU64(b.Locked, amount.Amount)
	if err != nil {
		return err
	}
	b.Balance = balance
	b.Locked = locked
	return nil
}

// Unlock subs an Amount from Balance.Locked until reach zero.
func (b *Balance) Unlock(amount Amount) error {
	locked, subbed := math.CheckedSubU64ToZero(b.Locked, amount.Amount)
	balance, err := math.CheckedAddU64(b.Balance, subbed)
	if err != nil {
		return err
	}
	b.Balance = balance
	b.Locked = locked
	return nil
}

// Supply an Amount from Balance.Balance to Balance.Supplied.
// Returns an error if not sufficient Balance.Balance.
func (b *Balance) Supply(amount Amount) error {
	balance, err := math.CheckedSubU64(b.Balance, amount.Amount)
	if err != nil {
		return err
	}
	supplied, err := math.CheckedAddU64(b.Supplied, amount.Amount)
	if err != nil {
		return err
	}
	b.Balance = balance
	b.Supplied = supplied
	return nil
}

// Withdraw an Amount from Balance.Supplied to Balance.Balance.
// Returns an error if not sufficient Balance.Supplied.
func (b *Balance) Withdraw(amount Amount) error {
	supplied, err := math.CheckedSubU64(b.Supplied, amount.Amount)
	if err != nil {
		return err
	}
	balance, err := math.CheckedAddU64(b.Balance, amount.Amount)
	if err != nil {
		return err
	}
	b.Balance = balance
	b.Supplied = supplied
	return nil
}

// Borrow an Amount from Leverage module and adds it to Balance.Balance and Balance.Borrowed.
func (b *Balance) Borrow(amount Amount, isVirtual bool) error {
	balance, err := math.CheckedAddU64(b.Balance, amount.Amount)
	if err != nil {
		return err
	}

	if isVirtual {
		virtual, err := math.CheckedAddU64(b.Virtual, amount.Amount)
		if err != nil {
			return err
		}
		b.Virtual = virtual
	} else {
		borrowed, err := math.CheckedAddU64(b.Borrowed, amount.Amount)
		if err != nil {
			return err
		}
		b.Borrowed = borrowed
	}

	b.Balance = balance

	return nil
}

// Repay an Amount from Balance.Borrowed.
// Returns an error if not sufficient Balance.Borrowed.
func (b *Balance) Repay(amount Amount) error {
	borrowed, err := math.CheckedSubU64(b.Borrowed, amount.Amount)
	if err != nil {
		return err
	}
	b.Borrowed = borrowed
	return nil
}

func (b *Balance) String() string {
	return fmt.Sprintf("%#v", *b)
}
