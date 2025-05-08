package kafka

import (
	"context"
	"fmt"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// ConsumerLag holds lag information for a specific topic-partition.
type ConsumerLag struct {
	Topic           string
	Partition       int32
	CommittedOffset int64
	LogEndOffset    int64
	Lag             int64
}

// CollectConsumerLag retrieves lag metrics for all topic-partitions in the specified consumer group.
func CollectConsumerLag(ctx context.Context, client *kgo.Client, group string) ([]ConsumerLag, error) {
	admin := kadm.NewClient(client)
	defer admin.Close()

	// Fetch committed offsets for the consumer group
	offsetsResp, err := admin.FetchOffsets(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch committed offsets: %w", err)
	}

	// Prepare a map to hold topic-partition pairs
	topics := make([]string, 0, len(offsetsResp))
	for tp := range offsetsResp {
		topics = append(topics, tp)
	}

	// Fetch log end offsets for the topic-partitions
	endOffsetsResp, err := admin.ListEndOffsets(ctx, topics...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch log end offsets: %w", err)
	}

	// Compile lag information
	var lags []ConsumerLag
	for tp, partitionOffset := range offsetsResp {
		for partition, committed := range partitionOffset {
			endOffset, exists := endOffsetsResp[tp][partition]
			if !exists {
				continue
			}

			lag := endOffset.Offset - committed.Offset.At
			if lag < 0 {
				lag = 0
			}

			lags = append(
				lags, ConsumerLag{
					Topic:           tp,
					Partition:       partition,
					CommittedOffset: committed.Offset.At,
					LogEndOffset:    endOffset.Offset,
					Lag:             lag,
				},
			)
		}
	}

	return lags, nil
}
