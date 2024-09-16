package cache

import (
	"github.com/dora-network/dora-service-utils/kafka"
	"github.com/rs/zerolog"
	"github.com/twmb/franz-go/pkg/kgo"
	"os"
	"time"
)

type options[K comparable, V any] struct {
	config         kafka.Config
	consumeTopics  []string
	consumerGroup  string
	pollTimeout    time.Duration
	maxPollRecords int
	userID         string
	password       string
	processFunc    ProcessRecordsFunc[K, V]
	logger         zerolog.Logger
	client         kafka.Client
}

type Option[K comparable, V any] func(options[K, V]) options[K, V]

type ProcessRecordsFunc[K comparable, V any] func(fetches kgo.Fetches, cache *map[K]V) error

// WithKafkaConfig sets the Kafka configuration for the cache.
func WithKafkaConfig[K comparable, V any](config kafka.Config) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.config = config
		return o
	}
}

// WithBrokers sets the Kafka brokers for the cache.
func WithBrokers[K comparable, V any](brokers ...string) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.config.Brokers = brokers
		return o
	}
}

// WithConsumeTopics sets the topics to consume from Kafka.
func WithConsumeTopics[K comparable, V any](topics ...string) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.consumeTopics = topics
		return o
	}
}

// WithConsumerGroup sets the consumer group for the Kafka client to use when consuming
// from Kafka topics.
func WithConsumerGroup[K comparable, V any](consumerGroup string) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.consumerGroup = consumerGroup
		return o
	}
}

// WithPollTimeout sets the poll timeout for the Kafka client.
func WithPollTimeout[K comparable, V any](timeout time.Duration) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.pollTimeout = timeout
		return o
	}
}

// WithMaxPollRecords sets the maximum number of records to poll from Kafka.
func WithMaxPollRecords[K comparable, V any](records int) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.maxPollRecords = records
		return o
	}
}

// WithUserID sets the user ID for the SASL plain authentication.
func WithUserID[K comparable, V any](userID string) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.userID = userID
		return o
	}
}

// WithPassword sets the password for the SASL plain authentication.
func WithPassword[K comparable, V any](password string) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.password = password
		return o
	}
}

// WithProcessFunc sets the function to process records fetched from Kafka.
func WithProcessFunc[K comparable, V any](processFunc ProcessRecordsFunc[K, V]) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.processFunc = processFunc
		return o
	}
}

// WithLogger sets the logger to use for the cache agent.
func WithLogger[K comparable, V any](logger zerolog.Logger) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.logger = logger
		return o
	}
}

// WithClient sets the Kafka client for the cache. Useful for testing.
func WithClient[K comparable, V any](client kafka.Client) Option[K, V] {
	return func(o options[K, V]) options[K, V] {
		o.client = client
		return o
	}
}

func defaultOptions[K comparable, V any]() options[K, V] {
	opts := options[K, V]{
		config:      kafka.DefaultConfig(),
		pollTimeout: time.Second,
		// This should be overridden by the development team with an appropriate handler function
		processFunc: processRecordsFunc[K, V],
		// Default logger writes to Stderr
		logger: zerolog.New(os.Stderr),
	}
	return opts
}

// The very most basic function possible in case we forget to provide a process function
// we simply log out the record we're processing to Stderr
func processRecordsFunc[K comparable, V any](fetches kgo.Fetches, _ *map[K]V) error {
	logger := zerolog.New(os.Stderr)
	for _, fetch := range fetches {
		for _, topic := range fetch.Topics {
			for _, partition := range topic.Partitions {
				if partition.Err != nil {
					logger.Error().
						Str("topic", topic.Topic).
						Int32("partition", partition.Partition).
						Err(partition.Err).
						Msg("Error fetching records")

					continue
				}
				for _, record := range partition.Records {
					logger.Info().
						Str("topic", topic.Topic).
						Int32("partition", partition.Partition).
						Bytes("key", record.Key).
						Bytes("value", record.Value).
						Msg("Processing record")
				}
			}
		}
	}
	return nil
}
