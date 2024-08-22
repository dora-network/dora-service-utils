package balance

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
	Amount    uint64    `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

type Balances struct {
	UserID     string  `json:"user_id" redis:"user_id"`
	AssetID    string  `json:"asset_id" redis:"asset_id"`
	Balance    Balance `json:"balances" redis:"balances"`
	Borrowed   Balance `json:"borrowed" redis:"borrowed"`
	Collateral Balance `json:"collateral" redis:"collateral"`
	Supplied   Balance `json:"supplied" redis:"supplied"`
	Virtual    Balance `json:"virtual" redis:"virtual"`
}

func (b *Balances) MarshalBinary() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Balances) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b)
}

func UserBalanceKey(userID string) string {
	return fmt.Sprintf("balances:users:%s", userID)
}

func ModuleBalanceKey() string {
	return fmt.Sprintf("balances:modules")
}

func GetUserBalances(ctx context.Context, rdb redis.Client, timeout time.Duration, userIDs []string, assets ...string) ([]Balances, error) {
	watch := make([]string, len(userIDs))
	for i, userID := range userIDs {
		watch[i] = UserBalanceKey(userID)
	}
	return getBalance(ctx, rdb, timeout, watch, assets...)
}

func GetModuleBalances(ctx context.Context, rdb redis.Client, timeout time.Duration, assetIDs ...string) ([]Balances, error) {
	watch := []string{ModuleBalanceKey()}
	return getBalance(ctx, rdb, timeout, watch, assetIDs...)
}

func getBalance(ctx context.Context, rdb redis.Client, timeout time.Duration, keys []string, ids ...string) ([]Balances, error) {
	var balances []Balances

	f := func(tx *redisv9.Tx) error {
		// This is just a simple read from Redis, but we're going to read it in a transaction to ensure
		// that if some other process is changing the data while we are attempting to read it, we're not
		// reading it with stale data.

		// we use the TxPipelined method to execute multiple commands in a single transaction
		// and collect the results, if any of the keys we are watching have been modified
		// since we started the transaction, the transaction will fail and we will retry
		cmd, err := tx.TxPipelined(ctx, func(pipe redisv9.Pipeliner) error {
			for _, key := range keys {
				pipe.HMGet(ctx, key, ids...)
			}
			return nil
		})

		for _, c := range cmd {
			res, err := c.(*redisv9.SliceCmd).Result()
			if err != nil {
				return err
			}

			for _, v := range res {
				if v == nil {
					balances = append(balances, Balances{})
					continue
				}

				b := new(Balances)
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

func UpdateBalances(ctx context.Context, rdb redis.Client, txFunc func(tx *redisv9.Tx) error, timeout time.Duration, watch ...string) error {
	return redis.TryTransaction(
		ctx,
		rdb,
		txFunc,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		watch...,
	)
}
