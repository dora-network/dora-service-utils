package integration_test

import (
	"context"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/dora-network/dora-service-utils/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"testing"
	"time"
)

var (
	timeout = time.Second * 5
)

func TestDoraNetwork(t *testing.T) {
	dn, err := integration.NewDoraNetwork(t)
	require.NoError(t, err)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	summary, err := cli.NetworkList(ctx, network.ListOptions{Filters: filters.NewArgs(
		filters.Arg("id", dn.Network.ID),
	)})
	require.NoError(t, err)
	assert.Len(t, summary, 1)

	require.NoError(t, dn.Cleanup())
	summary, err = cli.NetworkList(ctx, network.ListOptions{Filters: filters.NewArgs(
		filters.Arg("id", dn.Network.ID),
	)})
	require.NoError(t, err)
	assert.Len(t, summary, 0)
}

func TestDoraNetwork_CreateKafkaResource(t *testing.T) {
	dn, err := integration.NewDoraNetwork(t)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, dn.Cleanup())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err = dn.CreateKafkaResource(t, ctx)
	require.NoError(t, err)
	assert.NotNil(t, dn.KafkaResource)

	// create a message and send it to the kafka topic
	kClient, err := dn.GetKafkaClient()
	require.NoError(t, err)
	kClient.AddConsumeTopics("test-topic")
	defer kClient.Close()

	err = kClient.ProduceSync(ctx, &kgo.Record{
		Topic: "test-topic",
		Value: []byte("hello world"),
	}).FirstErr()

	require.NoError(t, err)
	count := 0
	for {
		fetches := kClient.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				t.Logf("Error fetching records: %v", err)
			}
		}
		//assert.Len(t, fetches.Errors(), 0)
		iter := fetches.RecordIter()
		for !iter.Done() {
			count++
			record := iter.Next()
			elapsed := time.Since(record.Timestamp)
			t.Logf("Received message: %s, elapsed: %v", string(record.Value), elapsed)
			assert.Equal(t, "hello world", string(record.Value))
		}

		if count > 0 {
			break
		}
	}
	assert.Equal(t, 1, count)
}

func TestDoraNetwork_CreateRedisResource(t *testing.T) {
	dn, err := integration.NewDoraNetwork(t)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, dn.Cleanup())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err = dn.CreateRedisResource(t, ctx)
	require.NoError(t, err)

	rdb, err := dn.GetRedisClient()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rdb.Close())
	}()
	rdb.Set(ctx, "key", "value", 0)
	val, err := rdb.Get(ctx, "key").Result()
	require.NoError(t, err)
	assert.Equal(t, "value", val)
}
