package helpers

import (
	"errors"
	"fmt"

	"github.com/dora-network/dora-service-utils/ledger/types"
)

// ModulePositionTracker caches and records changes to module borrowed and balance, and exposes methods for borrowing
type ModulePositionTracker struct {
	module *types.Module
}

func NewModuleBorrowTracker(
	module *types.Module,
) *ModulePositionTracker {
	return &ModulePositionTracker{
		// all balances below start with modified = false
		module: module.Copy(),
	}
}

// Module returns a safe copy of mbt.Module
func (mbt *ModulePositionTracker) Module() *types.Module {
	return mbt.module.Copy()
}

// IsModified returns true if mbt.Module IsModified
func (mbt *ModulePositionTracker) IsModified() bool {
	return mbt.module.IsModified()
}

// modifyModuleBalance adds and subtracts from module balance.
// Error if result would be negative. Balance tracker is not mutated on error.
func (mbt *ModulePositionTracker) modifyModuleBalance(add, remove *types.Balances) error {
	newBals := mbt.module.Balance.AddBals(add).SubBals(remove)
	if newBals.HasNegative() {
		return errors.New("negative module balance is not permitted")
	}
	mbt.module.Balance = newBals
	return nil
}

// modifyModuleSupplied adds and subtracts from module supplied.
// Error if result would be negative. Balance tracker is not mutated on error.
func (mbt *ModulePositionTracker) modifyModuleSupplied(add, remove *types.Balances) error {
	newSupplied := mbt.module.Supplied.AddBals(add).SubBals(remove)
	if newSupplied.HasNegative() {
		return errors.New("negative module supplied is not permitted")
	}
	mbt.module.Supplied = newSupplied
	return nil
}

// modifyModuleBorrowed adds and subtracts from module borrowed.
// Error if result would be negative. Balance tracker is not mutated on error.
func (mbt *ModulePositionTracker) modifyModuleBorrowed(add, remove *types.Balances) error {
	newBorrowed := mbt.module.Borrowed.AddBals(add).SubBals(remove)
	if newBorrowed.HasNegative() {
		return errors.New("negative module borrowed is not permitted")
	}
	mbt.module.Borrowed = newBorrowed
	return nil
}

// modifyModuleVirtualBonds adds and subtracts from module virtual bonds.
// Error if result would be negative. Balance tracker is not mutated on error.
func (mbt *ModulePositionTracker) modifyModuleVirtualBonds(add, remove *types.Balances) error {
	newVirtual := mbt.module.Virtual.AddBals(add).SubBals(remove)
	if newVirtual.HasNegative() {
		return errors.New("negative module borrowed is not permitted")
	}
	mbt.module.Virtual = newVirtual
	return nil
}

// Borrow adds to the module virtual borrowed, or moves assets from module balance to borrowed,
// depending on asset usage.
func (mbt *ModulePositionTracker) Borrow(assets AssetData, asset *types.Balance) error {
	if asset.IsZero() {
		return nil
	}
	if !asset.Valid() {
		return fmt.Errorf("invalid asset: %s", asset)
	}
	if !assets.CanBorrow(asset.Asset) {
		return fmt.Errorf("asset %s cannot be borrowed", asset)
	}
	virtual := assets.IsVirtualBorrow(asset.Asset)
	borrow := types.NewBalances(asset.Asset, asset.Amt())
	if virtual {
		return mbt.modifyModuleVirtualBonds(borrow, types.EmptyBalances())
	} else {
		if err := mbt.modifyModuleBorrowed(borrow, types.EmptyBalances()); err != nil {
			return err
		}
		return mbt.modifyModuleBalance(types.EmptyBalances(), borrow)
	}
}

// Supply adds to the module balance and supplied.
func (mbt *ModulePositionTracker) Supply(asset *types.Balance) error {
	if !asset.Valid() {
		return fmt.Errorf("invalid asset: %s", asset)
	}
	supply := types.NewBalances(asset.Asset, asset.Amt())
	if err := mbt.modifyModuleBalance(supply, types.EmptyBalances()); err != nil {
		return err
	}
	if err := mbt.modifyModuleSupplied(supply, types.EmptyBalances()); err != nil {
		return err
	}
	return nil
}

func (mbt *ModulePositionTracker) CanBorrow(assets AssetData, bal *types.Balance) bool {
	if bal.IsZero() {
		return true
	}
	if !assets.CanBorrow(bal.Asset) {
		return false
	}
	if !assets.IsVirtualBorrow(bal.Asset) {
		// Direct borrow: module must have the balance
		// TODO: supply utilization limit
		return mbt.module.Balance.HasAtLeast(bal.Asset, bal.Amt())
	}
	// Collect module totals for virtual borrow limit
	var sumAllBorrows int64
	for _, id := range mbt.module.Borrowed.AssetIDs() {
		amt := mbt.module.Borrowed.AmountOf(id)
		sumAllBorrows += amt
	}
	for _, id := range mbt.module.Virtual.AssetIDs() {
		amt := mbt.module.Virtual.AmountOf(id)
		sumAllBorrows += amt
	}
	var sumAllSupplied int64
	for _, id := range mbt.module.Supplied.AssetIDs() {
		// TODO: partial contribution due to supply utilization limit
		amt := mbt.module.Supplied.AmountOf(id)
		sumAllSupplied += amt
	}
	// Virtual borrow: sum of all borrows must not exceed sum of all supplied
	// TODO: apply differing exponents of supplied and borrowed assets
	return sumAllSupplied >= sumAllBorrows+bal.Amt()
}
