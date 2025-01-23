package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/redis"
	redisv9 "github.com/redis/go-redis/v9"
	"time"
)

func UserPositionKey(userID string) string {
	return fmt.Sprintf("positions:users:%s", userID)
}

func ModulePositionKey() string {
	return fmt.Sprintf("positions")
}

func GetUsersPosition(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	userIDs ...string,
) (map[string]*types.Position, error) {
	watch := make([]string, len(userIDs))
	for i, userID := range userIDs {
		watch[i] = UserPositionKey(userID)
	}
	return getUsersPosition(ctx, rdb, timeout, watch)
}

func GetUsersPositionCmd(
	ctx context.Context,
	tx redis.Cmdable,
	userIDs ...string,
) ([]redisv9.Cmder, []string, error) {
	watch := make([]string, len(userIDs))
	for i, userID := range userIDs {
		watch[i] = UserPositionKey(userID)
	}
	cmds, err := tx.TxPipelined(
		ctx, func(pipe redisv9.Pipeliner) error {
			for _, key := range watch {
				pipe.HGetAll(ctx, key)
			}
			return nil
		},
	)

	return cmds, watch, err
}

func SetUsersPosition(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	reqs map[string]*types.Position,
) error {
	watch := make([]string, len(reqs))
	txFunc := func(tx *redisv9.Tx) error {
		for userID, position := range reqs {
			key := UserPositionKey(userID)
			watch = append(watch, key)
			values := make(map[string]any)
			values[position.UserID] = position

			// write the position to redis
			err := tx.HSet(ctx, key, values).Err()
			if err != nil {
				return err
			}
		}

		return nil
	}

	return redis.TryTransaction(
		ctx,
		rdb,
		txFunc,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		watch...,
	)
}

func SetUsersPositionCmd(ctx context.Context, tx redis.Cmdable, reqs map[string]*types.Position) (
	[]redisv9.Cmder,
	[]string,
) {
	watch := make([]string, 0, len(reqs))
	cmds := make([]redisv9.Cmder, 0, len(reqs))
	for userID, position := range reqs {
		key := UserPositionKey(userID)
		watch = append(watch, key)
		values := make(map[string]any)
		values[position.UserID] = position

		// write the position to redis
		cmd := tx.HSet(ctx, key, values)
		cmds = append(cmds, cmd)
	}

	return cmds, watch
}

func GetModulePosition(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
) (*types.Position, error) {
	return getModulePosition(ctx, rdb, timeout, ModulePositionKey())
}

func GetModulePositionCmd(ctx context.Context, tx redis.Cmdable) (*redisv9.SliceCmd, string) {
	return tx.HMGet(ctx, ModulePositionKey(), "module"), ModulePositionKey()
}

func SetModulePosition(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	position *types.Position,
) error {
	txFunc := func(tx *redisv9.Tx) error {
		values := make(map[string]any)
		values["module"] = position

		// write the balances to redis
		err := tx.HSet(ctx, ModulePositionKey(), values).Err()
		if err != nil {
			return err
		}

		return nil
	}

	return redis.TryTransaction(
		ctx,
		rdb,
		txFunc,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		ModulePositionKey(),
	)
}

func SetModulePositionCmd(ctx context.Context, tx redis.Cmdable, position *types.Position) (redisv9.Cmder, string) {
	values := make(map[string]any)
	values["module"] = position

	// write the balances to redis
	cmd := tx.HSet(ctx, ModulePositionKey(), values)
	return cmd, ModulePositionKey()
}

func getModulePosition(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	modulePositionKey string,
) (*types.Position, error) {
	position := new(types.Position)

	f := func(tx *redisv9.Tx) error {
		err := tx.HGet(ctx, modulePositionKey, "module").Scan(position)
		if err != nil {
			if errors.Is(err, redisv9.Nil) {
				return nil
			}
			return err
		}

		return nil
	}

	if err := redis.TryTransaction(
		ctx,
		rdb,
		f,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		modulePositionKey,
	); err != nil {
		return nil, err
	}

	return position, nil
}

func getUsersPosition(ctx context.Context, rdb redis.Client, timeout time.Duration, keys []string) (
	map[string]*types.Position,
	error,
) {
	positions := make(map[string]*types.Position)

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

			for _, v := range res {
				p := new(types.Position)
				if err := p.UnmarshalBinary([]byte(v)); err != nil {
					return err
				}
				positions[p.UserID] = p
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

	return positions, nil
}
