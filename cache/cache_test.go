package cache_test

import (
	"context"
	"errors"
	"github.com/cenkalti/backoff/v4"
	"github.com/dora-network/dora-service-utils/cache"
	"github.com/dora-network/dora-service-utils/kafka"
	"github.com/dora-network/dora-service-utils/kafka/kafkafakes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client := &kafkafakes.FakeClient{
		PollRecordsStub: func(ctx context.Context, i int) kgo.Fetches {
			return kgo.Fetches{
				{
					Topics: []kgo.FetchTopic{
						{
							Topic: "Some Topic",
							Partitions: []kgo.FetchPartition{
								{
									Records: []*kgo.Record{
										{
											Key:   []byte("key"),
											Value: []byte("value"),
										},
									},
								},
							},
						},
					},
				},
			}
		},
	}

	// Create a new cache
	testCache := cache.New[string, string](
		cache.WithClient[string, string](client),
		cache.WithProcessFunc[string, string](func(ctx context.Context, timeout time.Duration, client kafka.Client, fetches kgo.Fetches, cache *map[string]string) error {
			assert.Len(t, fetches, 1)
			assert.Len(t, fetches[0].Topics, 1)
			assert.Len(t, fetches[0].Topics[0].Partitions, 1)
			assert.Len(t, fetches[0].Topics[0].Partitions[0].Records, 1)

			for _, f := range fetches {
				for _, t := range f.Topics {
					for _, p := range t.Partitions {
						for _, r := range p.Records {
							(*cache)[string(r.Key)] = string(r.Value)
						}
					}
				}
			}

			return nil
		}),
	)

	t.Run("Cache should not be ready after creation", func(t *testing.T) {
		require.False(t, testCache.Ready())
	})

	t.Run("Cache should not be startable without initialization", func(t *testing.T) {
		assert.Error(t, testCache.Start(ctx))
	})

	t.Run("Cache should be ready after initialization", func(t *testing.T) {
		require.NoError(t, testCache.Init())
		assert.True(t, testCache.Ready())
	})

	t.Run("Cache should be startable after initialization", func(t *testing.T) {
		assert.NoError(t, testCache.Start(ctx))

		retryFn := func() error {
			if testCache.Status() != cache.StatusRunning {
				return errors.New("cache not running")
			}
			return nil
		}

		require.NoError(t, backoff.Retry(retryFn, backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond*100), 3)))
		assert.Equal(t, cache.StatusRunning, testCache.Status())
	})

	t.Run("Cache should not be startable if already running", func(t *testing.T) {
		require.Equal(t, cache.StatusRunning, testCache.Status())
		assert.Error(t, testCache.Start(ctx))
	})

	t.Run("Cache should have the correct records", func(t *testing.T) {
		assert.Len(t, testCache.Records(), 1)
		got, ok := testCache.Get("key")
		require.True(t, ok)
		assert.Equal(t, "value", got)
	})

	t.Run("Cache should be stoppable", func(t *testing.T) {
		assert.Equal(t, cache.StatusRunning, testCache.Status())
		testCache.Stop()
		assert.Equal(t, cache.StatusStopped, testCache.Status())
	})
}
