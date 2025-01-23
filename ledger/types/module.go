package types

import (
	"encoding/json"
	"github.com/dora-network/dora-service-utils/errors"
)

// Module contains a snapshot of all of the module's assets and debts.
type Module struct {
	// Assets owned (including bonds and currencies). Negative values indicate borrows.
	Balance Balances `json:"balances" redis:"balances"`
	// Assets supplied to module but not yet withdrawn.
	Supplied Balances `json:"supplied" redis:"supplied"`
	// Assets minted by virtual-borrowing but not yet repaid
	Virtual Balances `json:"virtual" redis:"virtual"`
	// Assets borrowed from supply but not yet repaid
	Borrowed Balances `json:"borrowed" redis:"borrowed"`

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
	return json.Unmarshal(data, m)
}

// Init sets a position's original field to its current json representation. No-op if already set.
// For a module object to be valid, this must be called after json unmarshaling.
func (m *Module) Init() {
	if m != nil && m.original == "" {
		j, err := json.Marshal(m)
		if err != nil {
			return
		}
		// Track exported fields for IsModified
		m.original = string(j)
		m.originalSequence = m.Sequence
	}
}

func InitialModule() Module {
	m := Module{
		// Balances (can Validate)
		Balance:  Balances{Bals: make(map[string]int64)},
		Supplied: Balances{Bals: make(map[string]int64)},
		Borrowed: Balances{Bals: make(map[string]int64)},
		Virtual:  Balances{Bals: make(map[string]int64)},
		// Tracking fields
		Sequence:    0,
		LastUpdated: 0,
	}
	// Track exported fields for IsModified, as well as initial Sequence
	m.Init()
	return m
}

func NewModule(
	balance, supplied, borrowed, virtual Balances,
	lastUpdated int64,
	sequence uint64,
) (Module, error) {
	m := Module{
		// Balances (can Validate)
		Balance:  balance,
		Supplied: supplied,
		Borrowed: borrowed,
		Virtual:  virtual,
		// Tracking fields
		Sequence:    sequence,
		LastUpdated: lastUpdated,
	}
	// Track exported fields for IsModified, as well as initial Sequence
	m.Init()
	return m, nil
}

// Validate that Module position does not contain any invalid or negative Balances
func (m Module) Validate() error {
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
	if m.original == "" {
		return errors.NewInternal("module position not initialized")
	}
	return nil
}

// String
func (m Module) String() string {
	j, _ := json.Marshal(m)
	return string(j)
}

// IsModified detects whether a module position's exported fields have changed at all since initialization.
func (m Module) IsModified() bool {
	return m.original != m.String()
}

// NextSequence returns module.originalSequence + 1
func (m Module) NextSequence() uint64 {
	return m.originalSequence + 1
}

// Copy entire module, including original and isModified data.
func (m Module) Copy() Module {
	j, _ := json.Marshal(m)
	module := Module{
		// Any unexported fields must be copied here, before unmarshal
		original:         m.original,
		originalSequence: m.originalSequence,
	}
	_ = json.Unmarshal(j, &module)
	return module
}

// Copy entire module, setting current state as original and isModified to false.
func (m Module) Snapshot() Module {
	j, _ := json.Marshal(m)
	module := Module{}
	_ = json.Unmarshal(j, &module)
	module.Init()
	return m
}
