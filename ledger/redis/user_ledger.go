package redis

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	redisv9 "github.com/redis/go-redis/v9"

	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/redis"
)

func GetUserLedger(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	userIDs ...string,
) ([]types.UserLedger, error) {
	watch := make([]string, len(userIDs))
	for i, userID := range userIDs {
		watch[i] = UserBalanceKey(userID)
	}
	return getUserLedger(ctx, rdb, timeout, watch)
}

func getUserLedger(ctx context.Context, rdb redis.Client, timeout time.Duration, keys []string) (
	[]types.UserLedger,
	error,
) {
	var ledgers []types.UserLedger

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
					pipe.HGetAll(ctx, key)
				}
				return nil
			},
		)

		for _, c := range cmd {
			res, err := c.(*redisv9.MapStringStringCmd).Result()
			if err != nil {
				return err
			}

			var balances []*types.Balance
			for _, v := range res {
				b := new(types.Balance)
				if err := b.UnmarshalBinary([]byte(v)); err != nil {
					return err
				}
				balances = append(balances, b)
			}
			if len(balances) > 0 {
				ledgers = append(ledgers, types.NewUserLedger(balances[0].UserID, balances...))
			} else {
				ledgers = append(ledgers, types.UserLedger{})
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

	return ledgers, nil
}
