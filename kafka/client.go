package kafka

import (
	"context"
	"github.com/twmb/franz-go/pkg/kgo"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Client
type Client interface {
	// Close closes the client.
	Close()
	// Ping checks brokers to see if any are available
	Ping(ctx context.Context) error
	// Produce sends a record to Kafka asynchronously, calling an optional promise with the record written to Kafka and potentially an error if the record failed.
	Produce(ctx context.Context, record *kgo.Record, promise func(*kgo.Record, error))
	// ProduceSync sends a record to Kafka and waits for it to be acknowledged.
	ProduceSync(ctx context.Context, record ...*kgo.Record) kgo.ProduceResults
	// PollRecords fetches records from Kafka.
	PollRecords(ctx context.Context, maxPollRecords int) kgo.Fetches
}
