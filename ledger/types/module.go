package types

import (
	"github.com/dora-network/dora-service-utils/errors"
	"github.com/goccy/go-json"
)

// Module contains a snapshot of all of the module's assets and debts.
type Module struct {
	// Assets owned (including bonds and currencies). Negative values indicate borrows.
	Balance *Balances `json:"balance" redis:"balance"`
	// Assets supplied to module but not yet withdrawn.
	Supplied *Balances `json:"supplied" redis:"supplied"`
	// Assets minted by virtual-borrowing but not yet repaid
	Virtual *Balances `json:"virtual" redis:"virtual"`
	// Assets borrowed from supply but not yet repaid
	Borrowed *Balances `json:"borrowed" redis:"borrowed"`
	// Assets provided to the module by LPs to fund coupon interest.
	CouponFunds *Balances `json:"coupon_funds" redis:"coupon_funds"`

	// Tracking fields - see position.go
	LastUpdated      int64  `json:"last_updated" redis:"last_updated"`
	Sequence         uint64 `json:"sequence" redis:"sequence"`
	original         string
	originalSequence uint64
}

func (m *Module) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m *Module) UnmarshalBinary(data []byte) error {
	err := json.Unmarshal(data, m)
	m.Init() // Init must be called after json unmarshaling
	return err
}

// Init sets a module's original field to its current json representation. No-op if already set.
// For a module object to be valid, this must be called after json unmarshaling.
func (m *Module) Init() {
	// Nil balances should not be allowed
	if m.Balance == nil {
		m.Balance = EmptyBalances()
	}
	if m.Supplied == nil {
		m.Supplied = EmptyBalances()
	}
	if m.Virtual == nil {
		m.Virtual = EmptyBalances()
	}
	if m.Borrowed == nil {
		m.Borrowed = EmptyBalances()
	}
	if m.CouponFunds == nil {
		m.CouponFunds = EmptyBalances()
	}
	// Store original state. No-op if already stored.
	if m.original == "" {
		j, err := json.Marshal(m)
		if err != nil {
			return
		}
		// Track exported fields for IsModified
		m.original = string(j)
		m.originalSequence = m.Sequence
	}
}

func InitialModule() *Module {
	m := &Module{
		// Tracking fields
		Sequence:    0,
		LastUpdated: 0,
		// Note: All *Balances fields are set to EmptyBalances by m.Init()
	}
	// Track exported fields for IsModified, as well as initial Sequence
	m.Init()
	return m
}

func NewModule(
	balance, supplied, borrowed, virtual, coupon *Balances,
	lastUpdated int64,
	sequence uint64,
) (*Module, error) {
	m := &Module{
		// Balances (can Validate)
		Balance:     balance,
		Supplied:    supplied,
		Borrowed:    borrowed,
		Virtual:     virtual,
		CouponFunds: coupon,
		// Tracking fields
		Sequence:    sequence,
		LastUpdated: lastUpdated,
	}
	// Track exported fields for IsModified, as well as initial Sequence
	// Also overrides any nil *Balances with EmptyBalances()
	m.Init()
	return m, m.Validate()
}

// Validate that Module position does not contain any invalid or negative Balances
func (m *Module) Validate() error {
	// Balances.Validate will panic if Balances are nil, so we check first
	if m.Balance == nil ||
		m.Supplied == nil ||
		m.Borrowed == nil ||
		m.Virtual == nil ||
		m.CouponFunds == nil {
		return errors.Data("nil Balances in Module")
	}
	if err := m.Balance.Validate(false); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "module Balance")
	}
	if err := m.Supplied.Validate(false); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "module Supplied")
	}
	if err := m.Virtual.Validate(false); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "module Virtual")
	}
	if err := m.Borrowed.Validate(false); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "module Borrowed")
	}
	if err := m.CouponFunds.Validate(false); err != nil {
		return errors.Wrap(errors.InvalidInputError, err, "module Coupon Funds")
	}
	if m.original == "" {
		return errors.NewInternal("module position not initialized")
	}
	return nil
}

// String
func (m *Module) String() string {
	j, _ := json.Marshal(m)
	return string(j)
}

// IsModified detects whether a module position's exported fields have changed at all since initialization.
func (m *Module) IsModified() bool {
	return m.original != m.String()
}

// NextSequence returns module.originalSequence + 1
func (m *Module) NextSequence() uint64 {
	return m.originalSequence + 1
}

// Copy entire module, including original and isModified data.
func (m *Module) Copy() *Module {
	j, _ := json.Marshal(m)
	module := &Module{
		// Any unexported fields must be copied here, before unmarshal
		original:         m.original,
		originalSequence: m.originalSequence,
	}
	_ = json.Unmarshal(j, module)
	return module
}

// Snapshot entire module, setting current state as original and isModified to false.
func (m *Module) Snapshot() *Module {
	j, _ := json.Marshal(m)
	module := Module{}
	_ = json.Unmarshal(j, &module)
	module.Init()
	return &module
}
