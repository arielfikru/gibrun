// Package gibrun provides an opinionated Redis framework for Go developers
// who prioritize acceleration, sustainability, and high-performance downstreaming.
//
// "Give Data. Run Fast." 🏃💨🥛
package gibrun

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Config holds the configuration for connecting to Redis.
// "Blueprint configuration" - transparent and straightforward.
type Config struct {
	// Addr is the Redis server address (e.g., "localhost:6379")
	Addr string
	// Password for Redis authentication. Leave empty for open transparency.
	Password string
	// DB is the Redis database number to use.
	DB int
}

// Client is the main gibrun client that wraps Redis operations
// with an opinionated, developer-friendly API.
type Client struct {
	rdb *redis.Client
}

// New creates a new gibrun Client with the given configuration.
// This is the "Pembentukan Kabinet" - initializing your Redis connection.
//
// Example:
//
//	app := gibrun.New(gibrun.Config{
//	    Addr:     "localhost:6379",
//	    Password: "",
//	    DB:       0,
//	})
func New(cfg Config) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &Client{
		rdb: rdb,
	}
}

// Ping checks the connection to Redis.
// Returns nil if the connection is healthy.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Close closes the Redis connection.
// Always defer this after creating a client.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Gib starts a data storage operation.
// "Gib" means to give - we're giving data to the cache.
//
// Example:
//
//	err := app.Gib(ctx, "key").Value(myStruct).TTL(5*time.Minute).Exec()
func (c *Client) Gib(ctx context.Context, key string) *GibBuilder {
	return &GibBuilder{
		ctx:    ctx,
		client: c,
		key:    key,
	}
}

// Run starts a data retrieval operation.
// Cepat, taktis, dan langsung kerja.
//
// Example:
//
//	found, err := app.Run(ctx, "key").Bind(&result)
func (c *Client) Run(ctx context.Context, key string) *RunBuilder {
	return &RunBuilder{
		ctx:    ctx,
		client: c,
		key:    key,
	}
}

// Sprint starts an atomic operation.
// For high-speed counter operations.
//
// Example:
//
//	newCount, _ := app.Sprint(ctx, "counter").Incr()
func (c *Client) Sprint(ctx context.Context, key string) *SprintBuilder {
	return &SprintBuilder{
		ctx:    ctx,
		client: c,
		key:    key,
	}
}

// Del deletes one or more keys from Redis.
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in Redis.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
