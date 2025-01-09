package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	redisv9 "github.com/redis/go-redis/v9"

	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/redis"
)

func UserBalanceKey(userID string) string {
	return fmt.Sprintf("balances:users:%s", userID)
}

func ModuleBalanceKey() string {
	return fmt.Sprintf("balances:modules")
}

func GetUserBalances(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	userIDs []string,
	assets ...string,
) ([]types.Balance, error) {
	watch := make([]string, len(userIDs))
	for i, userID := range userIDs {
		watch[i] = UserBalanceKey(userID)
	}
	return getBalances(ctx, rdb, timeout, watch, assets...)
}

func GetModuleBalances(ctx context.Context, rdb redis.Client, timeout time.Duration, assetIDs ...string) (
	[]types.Balance,
	error,
) {
	watch := []string{ModuleBalanceKey()}
	return getBalances(ctx, rdb, timeout, watch, assetIDs...)
}

func GetModuleBalancesCmd(
	ctx context.Context,
	tx redis.Cmdable,
	assetIDs ...string,
) ([]redisv9.Cmder, string, error) {
	watch := ModuleBalanceKey()
	cmd, err := tx.TxPipelined(
		ctx, func(pipe redisv9.Pipeliner) error {
			pipe.HMGet(ctx, watch, assetIDs...)
			return nil
		},
	)

	return cmd, watch, err
}

func getBalances(ctx context.Context, rdb redis.Client, timeout time.Duration, keys []string, ids ...string) (
	[]types.Balance,
	error,
) {
	var balances []types.Balance

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
					balances = append(balances, types.Balance{})
					continue
				}

				b := new(types.Balance)
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
	reqs map[string][]*types.Balance,
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
	bals []*types.Balance,
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

func SetBalancesCmd(ctx context.Context, tx redis.Cmdable, reqs map[string][]*types.Balance) ([]redisv9.Cmder, []string) {
	watch := make([]string, 0)
	cmds := make([]redisv9.Cmder, 0)
	for userID, bals := range reqs {
		key := UserBalanceKey(userID)
		watch = append(watch, key)
		values := make(map[string]any)
		for _, bal := range bals {
			values[bal.AssetID] = bal
		}

		// write the balances to redis
		cmd := tx.HSet(ctx, key, values)
		cmds = append(cmds, cmd)
	}
	return cmds, watch
}
