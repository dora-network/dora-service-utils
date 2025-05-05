package kafka

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
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
	// CommitRecords issues a synchronous offset commit for the offsets contained
	// within rs. Retryable errors are retried up to the configured retry limit,
	// and any unretryable error is returned.
	//
	// This function is useful as a simple way to commit offsets if you have
	// disabled autocommitting. As an alternative if you always want to commit
	// everything, see CommitUncommittedOffsets.
	//
	// Simple usage of this function may lead to duplicate records if a consumer
	// group rebalance occurs before or while this function is being executed. You
	// can avoid this scenario by calling CommitRecords in a custom
	// OnPartitionsRevoked, but for most workloads, a small bit of potential
	// duplicate processing is fine.  See the documentation on DisableAutoCommit
	// for more details. You can also avoid this problem by using
	// BlockRebalanceOnPoll, but that option comes with its own tradeoffs (refer to
	// its documentation).
	//
	// It is recommended to always commit records in order (per partition). If you
	// call this function twice with record for partition 0 at offset 999
	// initially, and then with record for partition 0 at offset 4, you will rewind
	// your commit.
	//
	// A use case for this function may be to partially process a batch of records,
	// commit, and then continue to process the rest of the records. It is not
	// recommended to call this for every record processed in a high throughput
	// scenario, because you do not want to unnecessarily increase load on Kafka.
	//
	// If you do not want to wait for this function to complete before continuing
	// processing records, you can call this function in a goroutine.
	CommitRecords(ctx context.Context, rs ...*kgo.Record) error
	// CommitUncommittedOffsets issues a synchronous offset commit for any
	// partition that has been consumed from that has uncommitted offsets.
	// Retryable errors are retried up to the configured retry limit, and any
	// unretryable error is returned.
	//
	// The recommended pattern for using this function is to have a poll / process
	// / commit loop. First PollFetches, then process every record, then call
	// CommitUncommittedOffsets.
	//
	// As an alternative if you want to commit specific records, see CommitRecords.
	CommitUncommittedOffsets(ctx context.Context) error
	// CommitMarkedOffsets issues a synchronous offset commit for any partition
	// that has been consumed from that has marked offsets.  Retryable errors are
	// retried up to the configured retry limit, and any unretryable error is
	// returned.
	//
	// This function is only useful if you have marked offsets with
	// MarkCommitRecords when using AutoCommitMarks, otherwise this is a no-op.
	//
	// The recommended pattern for using this function is to have a poll / process
	// / commit loop. First PollFetches, then process every record,
	// call MarkCommitRecords for the records you wish the commit and then call
	// CommitMarkedOffsets.
	//
	// As an alternative if you want to commit specific records, see CommitRecords.
	CommitMarkedOffsets(ctx context.Context) error
	// MarkCommitRecords marks records to be available for autocommitting. This
	// function is only useful if you use the AutoCommitMarks config option, see
	// the documentation on that option for more details. This function does not
	// allow rewinds.
	MarkCommitRecords(rs ...*kgo.Record)
}

type NewClientFunc func(config Config, produceTopic, consumerGroup string, consumeTopics ...string) (Client, error)

func NewClient(config Config, produceTopic, consumerGroup string, consumeTopics ...string) (Client, error) {
	opts := make([]kgo.Opt, 0)
	opts = append(opts, kgo.SeedBrokers(config.Brokers...))

	if produceTopic != "" {
		opts = append(opts, kgo.DefaultProduceTopic(produceTopic))
	}

	if consumerGroup != "" {
		opts = append(opts, kgo.ConsumerGroup(consumerGroup))
	}

	if len(consumeTopics) > 0 {
		opts = append(opts, kgo.ConsumeTopics(consumeTopics...))
	}

	if config.Authentication.Username != "" && config.Authentication.Password != "" {
		opts = append(opts, kgo.SASL(plain.Auth{
			User: config.Authentication.Username,
			Pass: config.Authentication.Password,
		}.AsMechanism()))
	}

	client, err := kgo.NewClient(
		opts...,
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}
