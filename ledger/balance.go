package ledger

import (
	"context"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"github.com/cenkalti/backoff/v4"

	"github.com/goccy/go-json"

	"github.com/dora-network/dora-service-utils/redis"
)

type Balance struct {
	UserID  string `json:"user_id" redis:"user_id"`
	AssetID string `json:"asset_id" redis:"asset_id"`
	// Available Balance
	Balance    Amount `json:"balance" redis:"balance"`
	Borrowed   Amount `json:"borrowed" redis:"borrowed"`
	Collateral Amount `json:"collateral" redis:"collateral"`
	Supplied   Amount `json:"supplied" redis:"supplied"`
	Virtual    Amount `json:"virtual" redis:"virtual"`
	Locked     Amount `json:"locked" redis:"locked"`
}

func (b *Balance) MarshalBinary() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Balance) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b)
}

func UserBalanceKey(userID string) string {
	return fmt.Sprintf("balances:users:%s", userID)
}

func ModuleBalanceKey() string {
	return fmt.Sprintf("balances:modules")
}

// IsZero returns true if all the Amount of the balance are zero.
func (b *Balance) IsZero() bool {
	if b == nil {
		return true
	}
	return b.Balance.IsZero() &&
		b.Borrowed.IsZero() &&
		b.Collateral.IsZero() &&
		b.Supplied.IsZero() &&
		b.Virtual.IsZero() &&
		b.Locked.IsZero()
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

	return b.Balance.Equal(x.Balance) &&
		b.Borrowed.Equal(x.Borrowed) &&
		b.Collateral.Equal(x.Collateral) &&
		b.Supplied.Equal(x.Supplied) &&
		b.Virtual.Equal(x.Virtual) &&
		b.Locked.Equal(x.Locked)
}

func NewBalance(userID, assetID string, balance, borrowed, collateral, supplied, virtual, locked uint64) *Balance {
	return &Balance{
		UserID:     userID,
		AssetID:    assetID,
		Balance:    NewAmount(assetID, balance),
		Borrowed:   NewAmount(assetID, borrowed),
		Collateral: NewAmount(assetID, collateral),
		Supplied:   NewAmount(assetID, supplied),
		Virtual:    NewAmount(assetID, virtual),
		Locked:     NewAmount(assetID, locked),
	}
}

func ZeroBalance(userID, assetID string) *Balance {
	return &Balance{
		UserID:     userID,
		AssetID:    assetID,
		Balance:    ZeroAmount(assetID),
		Borrowed:   ZeroAmount(assetID),
		Collateral: ZeroAmount(assetID),
		Supplied:   ZeroAmount(assetID),
		Virtual:    ZeroAmount(assetID),
		Locked:     ZeroAmount(assetID),
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

// Add an Amount to Balance.Balance.
func (b *Balance) Add(amount Amount) error {
	result, err := b.Balance.Add(amount)
	if err != nil {
		return err
	}
	b.Balance = result
	return nil
}

// Sub an Amount from Balance.Balance.
func (b *Balance) Sub(amount Amount) error {
	result, err := b.Balance.Sub(amount)
	if err != nil {
		return err
	}
	b.Balance = result
	return nil
}

// Lock adds an Amount to Balance.Locked. Returns an error if the result Balance.Locked
// is greater than the whole Balance.Balance
func (b *Balance) Lock(amount Amount) error {
	balance, err := b.Balance.Sub(amount)
	if err != nil {
		return err
	}
	locked, err := b.Locked.Add(amount)
	if err != nil {
		return err
	}
	b.Balance = balance
	b.Locked = locked
	return nil
}

// Unlock subs an Amount from Balance.Locked until reach zero.
func (b *Balance) Unlock(amount Amount) error {
	locked, subbed, err := b.Locked.SubToZero(amount)
	if err != nil {
		return err
	}
	balance, err := b.Balance.Add(subbed)
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
	balance, err := b.Balance.Sub(amount)
	if err != nil {
		return err
	}
	supplied, err := b.Supplied.Add(amount)
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
	supplied, err := b.Supplied.Sub(amount)
	if err != nil {
		return err
	}
	balance, err := b.Balance.Add(amount)
	if err != nil {
		return err
	}
	b.Balance = balance
	b.Supplied = supplied
	return nil
}

// Borrow an Amount from Leverage module and adds it to Balance.Balance and Balance.Borrowed.
func (b *Balance) Borrow(amount Amount, isVirtual bool) error {
	balance, err := b.Balance.Add(amount)
	if err != nil {
		return err
	}

	if isVirtual {
		virtual, err := b.Virtual.Add(amount)
		if err != nil {
			return err
		}
		b.Virtual = virtual
	} else {
		borrowed, err := b.Borrowed.Add(amount)
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
	borrowed, err := b.Borrowed.Sub(amount)
	if err != nil {
		return err
	}
	b.Borrowed = borrowed
	return nil
}

func (b *Balance) String() string {
	return fmt.Sprintf(
		"%s - %s - %s - %s - %s - %s - %s - %s",
		b.UserID,
		b.AssetID,
		b.Balance.String(),
		b.Borrowed.String(),
		b.Collateral.String(),
		b.Supplied.String(),
		b.Virtual.String(),
		b.Locked.String(),
	)
}

func GetUserBalances(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	userIDs []string,
	assets ...string,
) ([]Balance, error) {
	watch := make([]string, len(userIDs))
	for i, userID := range userIDs {
		watch[i] = UserBalanceKey(userID)
	}
	return getBalances(ctx, rdb, timeout, watch, assets...)
}

func GetModuleBalances(ctx context.Context, rdb redis.Client, timeout time.Duration, assetIDs ...string) (
	[]Balance,
	error,
) {
	watch := []string{ModuleBalanceKey()}
	return getBalances(ctx, rdb, timeout, watch, assetIDs...)
}

func getBalances(ctx context.Context, rdb redis.Client, timeout time.Duration, keys []string, ids ...string) (
	[]Balance,
	error,
) {
	var balances []Balance

	f := func(tx *redisv9.Tx) error {
		// This is just a simple read from Redis, but we're going to read it in a transaction to ensure
		// that if some other process is changing the data while we are attempting to read it, we're not
		// reading it with stale data.

		// we use the TxPipelined method to execute multiple commands in a single transaction
		// and collect the results, if any of the keys we are watching have been modified
		// since we started the transaction, the transaction will fail and we will retry
		cmd, err := tx.TxPipelined(
			ctx, func(pipe redisv9.Pipeliner) error {
				for _, key := range keys {
					pipe.HMGet(ctx, key, ids...)
				}
				return nil
			},
		)

		for _, c := range cmd {
			res, err := c.(*redisv9.SliceCmd).Result()
			if err != nil {
				return err
			}

			for _, v := range res {
				if v == nil {
					balances = append(balances, Balance{})
					continue
				}

				b := new(Balance)
				if err := b.UnmarshalBinary([]byte(v.(string))); err != nil {
					return err
				}
				balances = append(balances, *b)
			}
		}

		return err
	}

	if err := redis.TryTransaction(
		ctx,
		rdb,
		f,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		keys...,
	); err != nil {
		return nil, err
	}

	return balances, nil
}

func SetUserBalances(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	reqs map[string][]Balance,
) error {
	watch := make([]string, len(reqs))
	txFunc := func(tx *redisv9.Tx) error {
		for userID, bals := range reqs {
			key := UserBalanceKey(userID)
			watch = append(watch, key)
			values := make(map[string]any)
			for _, bal := range bals {
				values[bal.AssetID] = bal
			}

			// write the balances to redis
			err := tx.HSet(ctx, key, values).Err()
			if err != nil {
				return err
			}
		}

		return nil
	}

	return SetBalances(ctx, rdb, txFunc, timeout, watch...)
}

func SetModuleBalances(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	bals []Balance,
) error {
	watch := []string{ModuleBalanceKey()}
	txFunc := func(tx *redisv9.Tx) error {
		key := ModuleBalanceKey()
		values := make(map[string]any)
		for _, bal := range bals {
			values[bal.AssetID] = bal
		}

		// write the balances to redis
		err := tx.HSet(ctx, key, values).Err()
		if err != nil {
			return err
		}

		return nil
	}

	return SetBalances(ctx, rdb, txFunc, timeout, watch...)
}

func SetBalances(
	ctx context.Context,
	rdb redis.Client,
	txFunc func(tx *redisv9.Tx) error,
	timeout time.Duration,
	watch ...string,
) error {
	return redis.TryTransaction(
		ctx,
		rdb,
		txFunc,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		watch...,
	)
}
