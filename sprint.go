package gibrun

import (
	"context"
	"time"
)

// SprintBuilder provides a fluent API for atomic Redis operations.
// Optimized for high-speed counter and increment operations.
type SprintBuilder struct {
	ctx    context.Context
	client *Client
	key    string
}

// Incr increments the value by 1 and returns the new value.
// Creates the key with value 1 if it doesn't exist.
//
// Example:
//
//	newCount, err := app.Sprint(ctx, "counter:visitors").Incr()
func (b *SprintBuilder) Incr() (int64, error) {
	return b.client.rdb.Incr(b.ctx, b.key).Result()
}

// IncrBy increments the value by the specified amount.
// Creates the key with the given value if it doesn't exist.
//
// Example:
//
//	newCount, err := app.Sprint(ctx, "counter:score").IncrBy(10)
func (b *SprintBuilder) IncrBy(n int64) (int64, error) {
	return b.client.rdb.IncrBy(b.ctx, b.key, n).Result()
}

// Decr decrements the value by 1 and returns the new value.
// Creates the key with value -1 if it doesn't exist.
//
// Example:
//
//	newCount, err := app.Sprint(ctx, "counter:stock").Decr()
func (b *SprintBuilder) Decr() (int64, error) {
	return b.client.rdb.Decr(b.ctx, b.key).Result()
}

// DecrBy decrements the value by the specified amount.
//
// Example:
//
//	newCount, err := app.Sprint(ctx, "counter:balance").DecrBy(100)
func (b *SprintBuilder) DecrBy(n int64) (int64, error) {
	return b.client.rdb.DecrBy(b.ctx, b.key, n).Result()
}

// IncrByFloat increments the value by a float amount.
// Useful for precise decimal operations.
//
// Example:
//
//	newVal, err := app.Sprint(ctx, "price:btc").IncrByFloat(0.05)
func (b *SprintBuilder) IncrByFloat(n float64) (float64, error) {
	return b.client.rdb.IncrByFloat(b.ctx, b.key, n).Result()
}

// Get returns the current value as int64.
// Returns 0 if the key doesn't exist.
func (b *SprintBuilder) Get() (int64, error) {
	val, err := b.client.rdb.Get(b.ctx, b.key).Int64()
	if err != nil {
		// Key doesn't exist, return 0
		return 0, nil
	}
	return val, nil
}

// SetWithTTL sets the counter to a specific value with TTL.
// Useful for rate limiting scenarios.
//
// Example:
//
//	err := app.Sprint(ctx, "ratelimit:user:123").SetWithTTL(1, time.Minute)
func (b *SprintBuilder) SetWithTTL(value int64, ttl time.Duration) error {
	return b.client.rdb.Set(b.ctx, b.key, value, ttl).Err()
}

// Expire sets a TTL on an existing counter.
//
// Example:
//
//	err := app.Sprint(ctx, "temp:counter").Expire(time.Hour)
func (b *SprintBuilder) Expire(ttl time.Duration) error {
	return b.client.rdb.Expire(b.ctx, b.key, ttl).Err()
}
