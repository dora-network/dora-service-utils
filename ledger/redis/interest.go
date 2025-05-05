package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dora-network/dora-service-utils/helpers"
	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/redis"
	redisv9 "github.com/redis/go-redis/v9"
)

// copied from graphtypes, but added user and time fields
// TODO: Do we still use this format for txs?
type TxLendingInterestAccrual struct {
	User string
	Time int64

	FromUnixTime int    `json:"fromUnixTime" firestore:"fromUnixTime" graphql:"fromUnixTime"`
	ToUnixTime   int    `json:"toUnixTime" firestore:"toUnixTime" graphql:"toUnixTime"`
	Earned       string `json:"earned" firestore:"earned" graphql:"earned"`
	Owed         string `json:"owed" firestore:"owed" graphql:"owed"`
}

func UserInterestKey(userID string) string {
	return fmt.Sprintf("interest:users:%s", userID)
}

const (
	InterestEarned      = "earned"
	InterestOwed        = "owed"
	InterestClaimed     = "claimed"
	InterestPaid        = "paid"
	InterestLastUpdated = "last_updated"

	secondsPerYear = 31556952.0 // 365.2425 days in a year
	MODULE         = "module"
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
	var (
		interests []types.Interest
		err       error
	)

	f := func(tx *redisv9.Tx) error {
		interests, err = getInterestTx(ctx, tx, keys...)
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

func getInterestTx(ctx context.Context, tx redis.Cmdable, keys ...string) ([]types.Interest, error) {
	var interests []types.Interest

	cmd, err := tx.TxPipelined(ctx, func(pipe redisv9.Pipeliner) error {
		for _, key := range keys {
			pipe.HGetAll(ctx, key)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, c := range cmd {
		res, err := c.(*redisv9.MapStringStringCmd).Result()
		if err != nil {
			return nil, err
		}

		if len(res) == 0 {
			interests = append(interests, types.Interest{})
			continue
		}

		earned, err := strconv.ParseUint(res[InterestEarned], 10, 64)
		if err != nil {
			return nil, err
		}

		owed, err := strconv.ParseUint(res[InterestOwed], 10, 64)
		if err != nil {
			return nil, err
		}

		claimed, err := strconv.ParseUint(res[InterestClaimed], 10, 64)
		if err != nil {
			return nil, err
		}

		paid, err := strconv.ParseUint(res[InterestPaid], 10, 64)
		if err != nil {
			return nil, err
		}

		lastUpdated, err := time.Parse(time.RFC3339Nano, res[InterestLastUpdated])
		if err != nil {
			return nil, err
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

	return interests, nil
}

func SetUserInterest(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	reqs map[string]*types.Interest,
) error {
	watch := make([]string, len(reqs))
	for userID := range reqs {
		watch = append(watch, UserInterestKey(userID))
	}
	txFunc := func(tx *redisv9.Tx) error {
		for userID, interest := range reqs {
			if err := setUserInterestTx(ctx, tx, userID, interest); err != nil {
				return err
			}
		}
		return nil
	}

	return redis.TryTransaction(ctx, rdb, txFunc, backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)), watch...)
}

func setUserInterestTx(ctx context.Context, tx redis.Cmdable, userID string, interest *types.Interest) error {
	key := UserInterestKey(userID)
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
	return nil
}

func AccrueLendingInterest(ctx context.Context, rdb redis.Client, timeout time.Duration, userID string, assetData helpers.AssetData, flatRate float64) (*TxLendingInterestAccrual, error) {
	// this has to be calculated within one transaction, we don't want positions changing between reads etc.
	// as this could lead to inconsistencies
	watch := []string{
		UserInterestKey(userID),
		UserPositionKey(userID),
		UserInterestKey(MODULE),
		ModulePositionKey(),
	}

	var accrualTransaction *TxLendingInterestAccrual

	txFunc := func(tx *redisv9.Tx) error {
		transaction, err := accrueLendingInterestTx(ctx, tx, userID, assetData, flatRate)
		if err != nil {
			return err
		}

		accrualTransaction = transaction
		return nil
	}

	err := redis.TryTransaction(ctx, rdb, txFunc, backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)), watch...)
	if err != nil {
		return nil, err
	}

	return accrualTransaction, nil
}

func AccrueAllLendingInterest(ctx context.Context, rdb redis.Client, timeout time.Duration, assetData helpers.AssetData, flatRate float64, watch []string, users ...string) ([]TxLendingInterestAccrual, error) {
	watch = append(watch, UserInterestKey(MODULE), ModulePositionKey())
	var accrualTransactions []TxLendingInterestAccrual

	txFunc := func(tx *redisv9.Tx) error {
		for _, userID := range users {
			transaction, err := accrueLendingInterestTx(ctx, tx, userID, assetData, flatRate)
			if err != nil {
				return err
			}

			accrualTransactions = append(accrualTransactions, *transaction)
		}

		return nil
	}

	err := redis.TryTransaction(ctx, rdb, txFunc, backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)), watch...)
	if err != nil {
		return nil, err
	}

	return accrualTransactions, nil
}

func accrueLendingInterestTx(ctx context.Context, tx redis.Cmdable, userID string, assetData helpers.AssetData, flatRate float64) (*TxLendingInterestAccrual, error) {
	now := time.Now()

	// first get the module positions
	var (
		moduleInterest     types.Interest
		accrualTransaction TxLendingInterestAccrual
	)
	interest, err := getInterestTx(ctx, tx, MODULE)
	if err != nil {
		return nil, err
	}

	if len(interest) > 0 {
		moduleInterest = interest[0]
	}

	// then get the user's positions
	userPositionsCmd, err := GetUsersPositionCmd(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	userPositions := types.InitialPosition(userID)

	for _, cmd := range userPositionsCmd {
		res, err := cmd.(*redisv9.MapStringStringCmd).Result()
		if err != nil {
			return nil, err
		}

		for _, v := range res {
			p := new(types.Position)
			if err := p.UnmarshalBinary([]byte(v)); err != nil {
				return nil, err
			}
			if p.UserID == userID {
				userPositions = p
				break
			}
		}
	}

	borrowedValue, err := assetData.ExactBorrowedValue(userPositions)
	if err != nil {
		return nil, err
	}

	var userInterest types.Interest
	interest, err = getInterestTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	if len(interest) > 0 {
		userInterest = interest[0]
	}

	if userInterest.LastUpdated.After(time.Now()) {
		return nil, fmt.Errorf("last updated time is in the future")
	}

	interestAccrued := 0.0

	if borrowedValue > 0 && now.After(userInterest.LastUpdated) {
		// compute interest accrued
		yearsElapsed := float64(now.Unix()-userInterest.LastUpdated.Unix()) / secondsPerYear
		interestAccrued = yearsElapsed * flatRate * borrowedValue
		userInterest.Owed += uint64(interestAccrued)
		moduleInterest.Owed += uint64(interestAccrued)
	}

	if err := setUserInterestTx(ctx, tx, userID, &userInterest); err != nil {
		return nil, err
	}

	if err := setUserInterestTx(ctx, tx, MODULE, &moduleInterest); err != nil {
		return nil, err
	}

	accrualTransaction = TxLendingInterestAccrual{
		User:         userID,
		Time:         now.Unix(),
		FromUnixTime: int(userInterest.LastUpdated.Unix()),
		ToUnixTime:   int(now.Unix()),
		Earned:       "0",
		Owed:         strconv.FormatFloat(float64(userInterest.Owed), 'f', -1, 64),
	}

	return &accrualTransaction, nil
}
