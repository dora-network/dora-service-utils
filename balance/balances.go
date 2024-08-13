package balance

import (
	"context"
	"errors"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"github.com/cenkalti/backoff/v4"

	"github.com/goccy/go-json"

	"github.com/dora-network/dora-service-utils/redis"
	"github.com/govalues/decimal"
)

type Amount struct {
	Amount    decimal.Decimal `json:"amount"`
	Timestamp time.Time       `json:"timestamp"`
}

type Balances struct {
	User       Amount `json:"user"`
	Borrowed   Amount `json:"borrowed"`
	Collateral Amount `json:"collateral"`
	Supplied   Amount `json:"supplied"`
	Virtual    Amount `json:"virtual"`
}

func (b *Balances) MarshalBinary() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Balances) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b)
}

func UserBalanceKey(userID string) string {
	return fmt.Sprintf("balances:%s", userID)
}

func GetUserBalances(ctx context.Context, rdb redis.Client, timeout time.Duration, userID, assetID string) (*Balances, error) {
	watch := UserBalanceKey(userID)

	balances := new(Balances)

	f := func(tx *redisv9.Tx) error {
		// This is just a simple read from Redis, but we're going to read it in a transaction to ensure
		// that if some other process is changing the data while we are attempting to read it, we're not
		// reading it with stale data.
		bs, err := tx.HGet(ctx, UserBalanceKey(userID), assetID).Bytes()
		if err != nil {
			if errors.Is(err, redisv9.Nil) {
				return nil
			}

			return err
		}

		if err := balances.UnmarshalBinary(bs); err != nil {
			return err
		}

		return nil
	}

	if err := redis.TryTransaction(
		ctx,
		rdb,
		f,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		watch,
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
