package redis

import (
	"context"
	"github.com/cenkalti/backoff/v4"
	"github.com/dora-network/dora-service-utils/orderbook"
	"github.com/redis/go-redis/v9"
	"strings"
)

// SequenceNumberKey returns the key for the sequence number of a user.
func SequenceNumberKey(userID string) string {
	return Key(SequenceNumberPrefix, userID)
}

// OrderBookOrdersKey returns the key for retrieving an order book's open orders.
func OrderBookOrdersKey(orderBookID string) string {
	return Key(OrderBookPrefix, orderBookID)
}

// OrderBookPricesKey returns the key for retrieving the ordered prices of an order book.
func OrderBookPricesKey(orderBookID string, side orderbook.Side) string {
	return Key(OrderBookPrefix, orderBookID, "prices", string(side))
}

// OrdersAtPriceKey returns the key for retrieving the orders at a specific price in an order book.
func OrdersAtPriceKey(orderBookID string, side orderbook.Side, price string) string {
	return Key(OrderBookPrefix, orderBookID, price, string(side))
}

// OrderKey returns the key for retrieving an order.
func OrderKey(orderID string) string {
	return Key(OrderPrefix, orderID)
}

// BalancesKey returns the key for the balances of an entity.
func BalancesKey(entityID string) string {
	return Key(BalancesPrefix, entityID)
}

// Key constructs a redis key from the given elements. The elements should be provided in the
// order they should appear in the key. A key's format should follow the following pattern:
// - data type
// - record ID
// - additional distinguishing information
// for example: "order_book:baseAsset_quoteAsset:buy"
func Key(elems ...string) string {
	return strings.Join(elems, ":")
}

// TryTransaction retries the given transaction function until it succeeds or the backoff strategy gives up.
func TryTransaction(ctx context.Context, rdb Client, f func(tx *redis.Tx) error, backoffStrategy backoff.BackOff, keys ...string) error {
	retryFn := func() error {
		return rdb.Watch(ctx, f, keys...)
	}

	return backoff.Retry(retryFn, backoffStrategy)
}