package helpers

import (
	"math/big"
	"time"

	"github.com/rs/zerolog"

	"github.com/dora-network/bond-api-golang/graph/types"
	"github.com/dora-network/dora-service-utils/errors"
	ltypes "github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/math"
)

// OrderIterator supplies a stream of orders sharing the same type, and exposes methods for dealing with them.
// It can be used like an iterator, except calling next when the previous order has not yet been filled (or cancelled)
// will return that order again. If pool is non-nil, also contains the option to match with pool.
type OrderIterator struct {
	availableOrders []*types.Order  // Orders which have not yet been returned by Next()
	allOrders       []*types.Order  // All orders initially populated
	currentOrder    *types.Order    // Order most recently returned
	ordersModified  map[string]bool // Whether returned orders were actually modified

	// readMaxOrders is true if the iterator started with len(availableOrders) equal to the maximum
	// amount of orders it is allowed to process per tick (e.g. 25). If true, the iterator should be
	// considered exhausted when currentOrder is nil after Next, and there could have been more matching
	// orders in the database. If false, the iterator is considered done when currentOrder is nil after Next.
	readMaxOrders bool

	done      bool // True when Next() will no longer return any more orders
	exhausted bool // True when the iterator ran out of orders due to its read limit or an error

	// For logging reasons behind order cancellations
	logger zerolog.Logger
}

// AllUsers returns all users whose orders are present in iterator
func (oi OrderIterator) AllUsers() []string {
	users := []string{}
	found := map[string]bool{}
	for _, order := range oi.allOrders {
		userID := order.UserUID
		if !found[userID] {
			users = append(users, userID)
			found[userID] = true
		}
	}
	return users
}

// ModifiedOrders returns the subset of allOrders which were actually modified by Fill or Cancel
func (oi OrderIterator) ModifiedOrders() []*types.Order {
	orders := []*types.Order{}
	for _, o := range oi.allOrders {
		if oi.ordersModified[o.OrderID] {
			orders = append(orders, o)
		}
	}
	return orders
}

// IsOpen is true if an iterator's Next() will return a valid open order with nonzero amount left to fill.
func (oi *OrderIterator) IsOpen() bool {
	// Will be false if no orders remain, if iterator is at limit, if an invalid or closed error was next,
	// if next order somehow had zero remaining amount, or if order type was not in the types known by MaxFill.
	o := oi.Next()
	if o != nil && math.IsPositive(o.MaxFill()) {
		return true
	}
	return false
}

// IsExhausted is true if the order iterator has returned and filled the maximum amount of
// orders it was allowed to read from the database, or stopped on an invalid order or read error,
// but there may still be orders that would have matched its query.
func (oi *OrderIterator) IsExhausted() bool {
	_ = oi.Next() // To detect if the iterator will be exhausted on Next call, we do it here.
	return oi.exhausted
}

// MaxFill is the maximum amount of base asset the next order can fill.
// This is the remaining AmountIn for market sells and limit sells,
// or the remaining AmountOut for market buys and limit buys.
// Also returns zero if no orders remain in the iterator.
func (oi *OrderIterator) MaxFill() *big.Int {
	order := oi.Next()
	if order == nil || !order.State.IsOpen() {
		return big.NewInt(0)
	}
	remIn, remOut, err := order.GetRemainingAmts()
	if err != nil {
		return big.NewInt(0)
	}
	switch order.Type {
	case types.OrderTypeLimitSell, types.OrderTypeMarketSell:
		return remIn // Sells want to reach a target amountIn
	case types.OrderTypeLimitBuy, types.OrderTypeMarketBuy:
		return remOut // Buys want to reach a target amountOut
	}
	return big.NewInt(0)
}

// Next order to fill, or nil if done.
// Repeats the current order until it is executed or cancelled.
// This allows "if Next() != nil" to be used in control flow logic without skipping an order.
func (oi *OrderIterator) Next() *types.Order {
	if oi.done {
		// Once iterator ran out of orders once, it does not need to check again
		return nil
	}
	if oi.currentOrder != nil && math.IsPositive(oi.currentOrder.MaxFill()) {
		// If the order returned last time has yet to be fully filled, return it again
		return oi.currentOrder
	}
	// Sets oi.currentOrder to next available order, or nil
	oi.nextOrder()
	return oi.currentOrder
}

// nextOrder clears currentOrder then tries to set it to the next available order.
// sets oi.Done and oi.Exhausted when appropriate. No-op if already done.
// keep this function private. It is used by oi.Next()
func (oi *OrderIterator) nextOrder() {
	// No-op after done
	if oi.done {
		return
	}
	// Clear current order
	oi.currentOrder = nil
	// Set done if reached the end of available orders
	if len(oi.availableOrders) == 0 {
		oi.done = true
		oi.exhausted = oi.readMaxOrders // If there could have been more orders matching the query
		return
	}
	for len(oi.availableOrders) > 0 {
		// If there were available orders, move one to current orders
		oi.currentOrder = oi.availableOrders[0]
		oi.availableOrders = oi.availableOrders[1:]
		if math.IsPositive(oi.currentOrder.MaxFill()) {
			// If the new order was fillable, return immediately
			return
		} else {
			// Otherwise, try to cancel the order, and note whether it was modified
			reason := "MaxFill was not positive when selected by nextOrder"
			if oi.currentOrder.Cancel(reason) {
				oi.logCancel(oi.currentOrder.OrderID, oi.currentOrder.OrderBookID, reason, nil)
				oi.ordersModified[oi.currentOrder.OrderID] = true
			}
			oi.currentOrder = nil
			// Then try the next available order
			continue
		}
	}
	// Available orders exhausted by the above loop without returning
	oi.done = true
	oi.exhausted = oi.readMaxOrders // If there could have been more orders matching the query
}

// FillWithSwap the current order using given balances, and generate the appropriate swap transaction.
// This mutates the order, but not the pool or the user's balances. User balances must already be mutated to
// record user positions in transaction.
func (oi *OrderIterator) FillWithSwap(
	balanceIn, borrowed, balanceOut, swapFee *ltypes.Balance,
	users *UserPositionTracker,
	assets AssetData,
	isDoraV1 bool,
) (*types.Transaction, error) {
	order := oi.currentOrder
	if order == nil {
		return nil, errors.New(errors.InternalError, "orderIterator not ready to FillWithSwap")
	}
	if balanceIn.IsZero() || balanceOut.IsZero() {
		// error if amounts are zero. Upstream functions shouldn't even be calling this if balances are zero.
		return nil, errors.New(errors.InternalError, "zero balance input to FillWithSwap")
	}
	// Mutate order (and mark it executed if it was completely filled)
	bIn := types.NewBalance(balanceIn.Asset, balanceIn.Amt())
	bOut := types.NewBalance(balanceOut.Asset, balanceOut.Amt())
	if err := oi.currentOrder.Fill(bIn, bOut); err != nil {
		return nil, err
	}
	// Mark this order as modified
	oi.ordersModified[order.OrderID] = true
	// Compute price for tx
	executedPrice := math.ExecutedPrice(order.IsSell(), balanceIn.Amount, balanceOut.Amount)
	// mutate user balances
	if err := users.SwapBalance(order.UserUID, balanceIn, balanceOut, isDoraV1, order); err != nil {
		return nil, err
	}
	// Net stablecoin equivalance if possible
	users.CleanupStablecoinEquivalence(assets, order.UserUID)
	// Create Swap transaction (with Order ID)
	tx := types.NewTransaction(
		"", order.OrderBookID, order.UserUID, now(),
		users.InitialPosition(order.UserUID), users.FinalPosition(order.UserUID),
	).WithTxSwap(
		&types.TxSwap{
			BalanceIn:  bIn,
			BalanceOut: bOut,
			Borrowed:   types.NewBalance(borrowed.Asset, borrowed.Amt()),
			Price:      executedPrice,
			OrderID:    order.OrderID,
			Fees:       types.NewBalance(swapFee.Asset, swapFee.Amt()),
		},
	)
	return tx, nil
}

// FillWithCounterOrder the current order using given balances, and generate the appropriate match transaction.
// This mutates the order, but not the counter order or the user's balances. User balances must already be mutated
// in order to have the initial and final positions for the transaction.
func (oi *OrderIterator) FillWithCounterOrder(
	balanceIn, borrowed, balanceOut *ltypes.Balance,
	counterOrderID, counterUserID string,
	users *UserPositionTracker,
	assets AssetData,
	isDoraV1 bool,
) (*types.Transaction, error) {
	order := oi.currentOrder
	if order == nil {
		return nil, errors.NewInternal("orderIterator not ready to FillWithCounterOrder")
	}
	if balanceIn.IsZero() || balanceOut.IsZero() {
		// error if amounts are zero. Upstream functions shouldn't even be calling this if balances are zero.
		return nil, errors.NewInternal("zero balance input to FillWithCounterOrder")
	}
	// Mutate order (and mark it executed if it was completely filled)
	bIn := types.NewBalance(balanceIn.Asset, balanceIn.Amt())
	bOut := types.NewBalance(balanceOut.Asset, balanceOut.Amt())
	if err := order.Fill(bIn, bOut); err != nil {
		return nil, err
	}
	// Mark this order as modified
	oi.ordersModified[order.OrderID] = true
	// Compute price for tx
	executedPrice := math.ExecutedPrice(order.IsSell(), balanceIn.Amount, balanceOut.Amount)
	// mutate users balance
	if err := users.SwapBalance(order.UserUID, balanceIn, balanceOut, isDoraV1, order); err != nil {
		return nil, err
	}
	// Net stablecoin equivalance if possible
	users.CleanupStablecoinEquivalence(assets, order.UserUID)

	// Create MatchOrder transaction (with Order ID)
	tx := types.NewTransaction(
		"", order.OrderBookID, order.UserUID, now(),
		users.InitialPosition(order.UserUID), users.FinalPosition(order.UserUID),
	).WithTxMatchOrder(
		&types.TxMatchOrder{
			BalanceIn:      bIn,
			BalanceOut:     bOut,
			Borrowed:       types.NewBalance(borrowed.Asset, borrowed.Amt()),
			OrderID:        order.OrderID,
			CounterOrderID: counterOrderID,
			CounterUserID:  counterUserID,
			Price:          executedPrice,
		},
	)
	return tx, nil
}

// Cancel the current order, allowing Next to retrieve another order without filling the current one
func (oi *OrderIterator) Cancel(reason string, err error, users *UserPositionTracker) {
	if oi.currentOrder != nil {
		oi.logCancel(oi.currentOrder.OrderID, oi.currentOrder.OrderBookID, reason, err)
		modified := oi.currentOrder.Cancel(reason) // mutate order
		oi.ordersModified[oi.currentOrder.OrderID] = modified
		if modified {
			err = users.UnlockCanceledOrderBalance(oi.currentOrder)
			oi.logger.Err(err).Msgf("Error unlocking canceled order balance for user %s", oi.currentOrder.UserUID)
		}
	} else {
		oi.logCancel("nil", "nil", reason, err)
	}
}

// NewOrderIterator creates a new order iterator for a fixed set of orders in an orderbook.
// Set readMaxOrders to true if the orders passed in are at MATCH_ITER_LIMIT, so there could be more in db.
func NewOrderIterator(
	orders []*types.Order,
	logger zerolog.Logger,
	readMaxOrders bool,
) *OrderIterator {
	return &OrderIterator{
		availableOrders: orders,
		allOrders:       orders,
		readMaxOrders:   readMaxOrders,
		ordersModified:  map[string]bool{},
		logger:          logger,
	}
}

func now() int {
	return int(time.Now().Unix())
}

func (oi *OrderIterator) logCancel(orderID, orderbookID string, msg string, err error) {
	oi.logger.Info().
		Str("orderID", orderID).
		Str("orderbookID", orderbookID).
		Str("reason", msg).
		Err(err).
		Msgf("order cancelled")
}
