package cache

import (
	"context"
	"errors"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"sync"
)

// Cache is a generic in-memory cache that can be used to store data fetched from Kafka
// and processed by a user-defined function.
type Cache[K comparable, V any] struct {
	mu         sync.Mutex
	options    options[K, V]
	cancelFunc context.CancelFunc
	records    map[K]V
	status     Status
}

// New creates a new cache with the provided options.
func New[K comparable, V any](options ...Option[K, V]) *Cache[K, V] {
	opts := applyOptions[K, V](options...)
	return &Cache[K, V]{
		options: opts,
		records: make(map[K]V),
		status:  StatusNotReady,
	}
}

// Ready returns true if the cache is ready to be started.
func (c *Cache[K, V]) Ready() bool {
	return c.status == StatusReady
}

// Init initializes the cache by creating a new Kafka client with the provided options,
// or uses the client provided in the options.
func (c *Cache[K, V]) Init() error {
	// if the client has not been set then we should create a new client with the provided options
	if c.options.client != nil {
		c.status = StatusReady
		return nil
	}

	opts := make([]kgo.Opt, 0)
	opts = append(opts, kgo.SeedBrokers(c.options.config.Brokers...))
	opts = append(opts, kgo.ConsumeTopics(c.options.consumeTopics...))

	if c.options.consumerGroup != "" {
		opts = append(opts, kgo.ConsumerGroup(c.options.consumerGroup))
	}

	if c.options.userID != "" && c.options.password != "" {
		opts = append(opts, kgo.SASL(plain.Auth{
			User: c.options.userID,
			Pass: c.options.password,
		}.AsMechanism()))
	}
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return err
	}
	c.options.client = client
	c.status = StatusReady
	return nil
}

// Start starts the cache by polling records from Kafka and processing them using the
// user-defined function.
func (c *Cache[K, V]) Start(parent context.Context) error {
	if c.status == StatusNotReady {
		return errors.New("cache not initialized")
	}

	if c.status == StatusRunning {
		return errors.New("cache already running")
	}

	ctx, cancel := context.WithCancel(parent)
	c.cancelFunc = cancel
	go c.start(ctx)
	return nil
}

func (c *Cache[K, V]) start(ctx context.Context) {
	c.mu.Lock()
	c.status = StatusRunning
	c.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			pollCtx, cancel := context.WithTimeout(ctx, c.options.pollTimeout)
			fetches := c.options.client.PollRecords(pollCtx, c.options.maxPollRecords)
			cancel()
			// process records
			c.mu.Lock()
			if err := c.options.processFunc(ctx, c.options.processTimeout, fetches, &c.records); err != nil {
				c.options.logger.Error().Err(err).Msg("failed to process records")
			}
			c.mu.Unlock()
		}
	}
}

// Stop stops the cache from polling records from Kafka.
func (c *Cache[K, V]) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = StatusStopped
	c.cancelFunc()
}

// Status returns the current status of the cache.
func (c *Cache[K, V]) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// Records returns the records stored in the cache.
func (c *Cache[K, V]) Records() map[K]V {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.records
}

// Get returns the value stored in the cache for the provided key.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.records[key]
	return v, ok
}
