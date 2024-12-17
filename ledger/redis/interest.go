package redis

import (
	"context"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/redis"
	redisv9 "github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

func UserInterestKey(userID string) string {
	return fmt.Sprintf("interest:users:%s", userID)
}

const (
	InterestEarned      = "earned"
	InterestOwed        = "owed"
	InterestClaimed     = "claimed"
	InterestPaid        = "paid"
	InterestLastUpdated = "last_updated"
)

func GetUserInterest(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	userID ...string,
) ([]types.Interest, error) {
	watch := make([]string, len(userID))
	for i, id := range userID {
		watch[i] = UserInterestKey(id)
	}

	return getInterest(ctx, rdb, timeout, watch...)
}

func getInterest(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	keys ...string,
) ([]types.Interest, error) {
	var interests []types.Interest

	f := func(tx *redisv9.Tx) error {
		cmd, err := tx.TxPipelined(ctx, func(pipe redisv9.Pipeliner) error {
			for _, key := range keys {
				pipe.HGetAll(ctx, key)
			}
			return nil
		})

		for _, c := range cmd {
			res, err := c.(*redisv9.MapStringStringCmd).Result()
			if err != nil {
				return err
			}

			if len(res) == 0 {
				interests = append(interests, types.Interest{})
				continue
			}

			earned, err := strconv.ParseUint(res[InterestEarned], 10, 64)
			if err != nil {
				return err
			}

			owed, err := strconv.ParseUint(res[InterestOwed], 10, 64)
			if err != nil {
				return err
			}

			claimed, err := strconv.ParseUint(res[InterestClaimed], 10, 64)
			if err != nil {
				return err
			}

			paid, err := strconv.ParseUint(res[InterestPaid], 10, 64)
			if err != nil {
				return err
			}

			lastUpdated, err := time.Parse(time.RFC3339Nano, res[InterestLastUpdated])
			if err != nil {
				return err
			}

			interest := types.Interest{
				Earned:      earned,
				Owed:        owed,
				Claimed:     claimed,
				Paid:        paid,
				LastUpdated: lastUpdated,
			}

			interests = append(interests, interest)
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

	return interests, nil
}

func SetUserInterest(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	reqs map[string]*types.Interest,
) error {
	watch := make([]string, len(reqs))
	txFunc := func(tx *redisv9.Tx) error {
		for userID, interest := range reqs {
			key := UserInterestKey(userID)
			watch = append(watch, key)
			values := map[string]interface{}{
				InterestEarned:      interest.Earned,
				InterestOwed:        interest.Owed,
				InterestClaimed:     interest.Claimed,
				InterestPaid:        interest.Paid,
				InterestLastUpdated: interest.LastUpdated,
			}
			if _, err := tx.HMSet(ctx, key, values).Result(); err != nil {
				return err
			}
		}
		return nil
	}

	return redis.TryTransaction(ctx, rdb, txFunc, backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)), watch...)
}
