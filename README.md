# DORA Service utilities

A collection of utilities that can be used across all DORA services

## Usage

### Balances

The balances package provides helpers for working with balances that are stored in a Redis cache.
The Redis cache allows the individual services to access shared in-memory data that require atomic operations.

#### Get User Balances

The `GetUserBalances` function retrieves the balances for a user from the Redis cache.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/dora-network/dora-service-utils/balances"
	redisv9 "github.com/go-redis/redis/v9"
)

func main() {
    // Create a new Redis client
    rdb := redisv9.NewClient(&redisv9.Options{
        Addr: "localhost:6379",
        Password: "",
        DB: 0,
    })
    
    // Get the balances for a user
    userBalances, err := balances.GetUserBalances(context.Background(), rdb, "user1")
    if err != nil {
        log.Fatalf("failed to get user balances: %v", err)
    }

    fmt.Printf("User balances: %v\n", userBalances)
}
```

#### Update Balances

The `UpdateBalances` provides a Redis transaction wrapper for updating balances in the Redis cache. It takes a function
that takes a redis.Tx reference and returns an error, a retry timeout, and a list of keys to watch for optimistic locking.

The wrapper will attempt to execute the function until it succeeds or the retry timeout is reached. If any of the keys in the
watch list are modified by another transaction, the function will fail, and the transaction will be retried. This allows for
safe concurrent updates to the Redis cache.

```go

package main

```