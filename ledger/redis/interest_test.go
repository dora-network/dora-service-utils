package redis

import (
	"context"
	"github.com/dora-network/dora-service-utils/ledger/types"
	"github.com/dora-network/dora-service-utils/testing/integration"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
	"testing"
	"time"
)

func TestUserInterest_Redis(t *testing.T) {
	dn, err := integration.NewDoraNetwork(t)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, dn.Cleanup())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, dn.CreateRedisResource(t, ctx))

	rdb, err := dn.GetRedisClient()
	require.NoError(t, err)

	t.Run("should set the interest for a single user", func(tt *testing.T) {
		lastUpdated := time.Now()
		interest := map[string]*types.Interest{
			"user1": {
				Earned:      100,
				Owed:        200,
				Claimed:     300,
				Paid:        400,
				LastUpdated: lastUpdated,
			},
		}

		err := SetUserInterest(ctx, rdb, time.Second, interest)
		require.NoError(tt, err)

		key := UserInterestKey("user1")
		res, err := rdb.HGetAll(ctx, key).Result()
		require.NoError(tt, err)
		require.Len(tt, res, 5)
		assert.Equal(tt, res[InterestEarned], "100")
		assert.Equal(tt, res[InterestOwed], "200")
		assert.Equal(tt, res[InterestClaimed], "300")
		assert.Equal(tt, res[InterestPaid], "400")
		assert.Equal(tt, res[InterestLastUpdated], lastUpdated.Format(time.RFC3339Nano))
	})

	t.Run("should set the interest for multiple users", func(tt *testing.T) {
		lastUpdated := time.Now()
		interest := map[string]*types.Interest{
			"user1": {
				Earned:      100,
				Owed:        200,
				Claimed:     300,
				Paid:        400,
				LastUpdated: lastUpdated,
			},
			"user2": {
				Earned:      101,
				Owed:        201,
				Claimed:     301,
				Paid:        401,
				LastUpdated: lastUpdated,
			},
			"user3": {
				Earned:      102,
				Owed:        202,
				Claimed:     302,
				Paid:        402,
				LastUpdated: lastUpdated,
			},
		}

		err := SetUserInterest(ctx, rdb, time.Second, interest)
		require.NoError(tt, err)

		key := UserInterestKey("user1")
		res, err := rdb.HGetAll(ctx, key).Result()
		require.NoError(tt, err)
		require.Len(tt, res, 5)
		assert.Equal(tt, res[InterestEarned], "100")
		assert.Equal(tt, res[InterestOwed], "200")
		assert.Equal(tt, res[InterestClaimed], "300")
		assert.Equal(tt, res[InterestPaid], "400")
		assert.Equal(tt, res[InterestLastUpdated], lastUpdated.Format(time.RFC3339Nano))
		key = UserInterestKey("user2")
		res, err = rdb.HGetAll(ctx, key).Result()
		require.NoError(tt, err)
		require.Len(tt, res, 5)
		assert.Equal(tt, res[InterestEarned], "101")
		assert.Equal(tt, res[InterestOwed], "201")
		assert.Equal(tt, res[InterestClaimed], "301")
		assert.Equal(tt, res[InterestPaid], "401")
		assert.Equal(tt, res[InterestLastUpdated], lastUpdated.Format(time.RFC3339Nano))
		key = UserInterestKey("user3")
		res, err = rdb.HGetAll(ctx, key).Result()
		require.NoError(tt, err)
		require.Len(tt, res, 5)
		assert.Equal(tt, res[InterestEarned], "102")
		assert.Equal(tt, res[InterestOwed], "202")
		assert.Equal(tt, res[InterestClaimed], "302")
		assert.Equal(tt, res[InterestPaid], "402")
		assert.Equal(tt, res[InterestLastUpdated], lastUpdated.Format(time.RFC3339Nano))
	})

	t.Run("should return empty interest if the user record doesn't exist", func(tt *testing.T) {
		rdb.Del(ctx, UserInterestKey("user1"))
		rdb.Del(ctx, UserInterestKey("user2"))
		rdb.Del(ctx, UserInterestKey("user3"))

		interests, err := GetUserInterest(ctx, rdb, time.Second, "user1", "user2")
		require.NoError(tt, err)
		require.NotNil(tt, interests)
		emptyInterest := []types.Interest{
			{},
			{},
		}
		assert.DeepEqual(tt, emptyInterest, interests)
	})

	t.Run("should retrieve the users interest if it exists", func(tt *testing.T) {
		lastUpdated := time.Now()

		rdb.HSet(
			ctx,
			UserInterestKey("user1"),
			types.Interest{
				Earned:      100,
				Owed:        200,
				Claimed:     300,
				Paid:        400,
				LastUpdated: lastUpdated,
			},
		)

		rdb.HMSet(
			ctx,
			UserInterestKey("user2"),
			types.Interest{
				Earned:      102,
				Owed:        202,
				Claimed:     302,
				Paid:        402,
				LastUpdated: lastUpdated,
			},
		)

		rdb.HMSet(
			ctx,
			UserInterestKey("user2"),
			types.Interest{
				Earned:      102,
				Owed:        202,
				Claimed:     302,
				Paid:        402,
				LastUpdated: lastUpdated,
			},
		)

		interests, err := GetUserInterest(ctx, rdb, time.Second, "user1")
		require.NoError(tt, err)
		require.NotNil(tt, interests)
		want := []types.Interest{
			{
				Earned:      100,
				Owed:        200,
				Claimed:     300,
				Paid:        400,
				LastUpdated: lastUpdated,
			},
		}

		assert.DeepEqual(tt, want, interests)
	})

	t.Run("should retrieve the interest for multiple users", func(tt *testing.T) {
		lastUpdated := time.Now()

		rdb.HSet(
			ctx,
			UserInterestKey("user1"),
			InterestEarned, 100,
			InterestOwed, 200,
			InterestClaimed, 300,
			InterestPaid, 400,
			InterestLastUpdated, lastUpdated,
		)

		rdb.HSet(
			ctx,
			UserInterestKey("user2"),
			InterestEarned, 101,
			InterestOwed, 201,
			InterestClaimed, 301,
			InterestPaid, 401,
			InterestLastUpdated, lastUpdated,
		)

		rdb.HSet(
			ctx,
			UserInterestKey("user3"),
			InterestEarned, 102,
			InterestOwed, 202,
			InterestClaimed, 302,
			InterestPaid, 402,
			InterestLastUpdated, lastUpdated,
		)

		interests, err := GetUserInterest(ctx, rdb, time.Second, "user1", "user3")
		require.NoError(tt, err)
		require.NotNil(tt, interests)
		want := []types.Interest{
			{
				Earned:      100,
				Owed:        200,
				Claimed:     300,
				Paid:        400,
				LastUpdated: lastUpdated,
			},
			{
				Earned:      102,
				Owed:        202,
				Claimed:     302,
				Paid:        402,
				LastUpdated: lastUpdated,
			},
		}

		assert.DeepEqual(tt, want, interests)
	})
}
