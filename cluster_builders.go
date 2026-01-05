package gibrun

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// ClusterGibBuilder provides a fluent API for storing data in Redis Cluster.
type ClusterGibBuilder struct {
	ctx    context.Context
	client *ClusterClient
	key    string
	value  any
	ttl    time.Duration
}

// Value sets the data to be stored.
func (b *ClusterGibBuilder) Value(v any) *ClusterGibBuilder {
	b.value = v
	return b
}

// TTL sets the time-to-live for the cached data.
func (b *ClusterGibBuilder) TTL(d time.Duration) *ClusterGibBuilder {
	b.ttl = d
	return b
}

// Exec executes the storage operation.
func (b *ClusterGibBuilder) Exec() error {
	if b.value == nil {
		return ErrNilValue
	}

	data, err := b.marshal(b.value)
	if err != nil {
		return err
	}

	if b.ttl > 0 {
		return b.client.rdb.Set(b.ctx, b.key, data, b.ttl).Err()
	}
	return b.client.rdb.Set(b.ctx, b.key, data, 0).Err()
}

func (b *ClusterGibBuilder) marshal(v any) ([]byte, error) {
	switch val := v.(type) {
	case string:
		return []byte(val), nil
	case []byte:
		return val, nil
	default:
		return json.Marshal(val)
	}
}

// ClusterRunBuilder provides a fluent API for retrieving data from Redis Cluster.
type ClusterRunBuilder struct {
	ctx    context.Context
	client *ClusterClient
	key    string
}

// Bind retrieves the data and unmarshals it into the provided pointer.
func (b *ClusterRunBuilder) Bind(dest any) (bool, error) {
	if dest == nil {
		return false, ErrNilPointer
	}

	data, err := b.client.rdb.Get(b.ctx, b.key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	if err := b.unmarshal(data, dest); err != nil {
		return false, err
	}

	return true, nil
}

// Raw retrieves the raw string value without unmarshalling.
func (b *ClusterRunBuilder) Raw() (string, bool, error) {
	val, err := b.client.rdb.Get(b.ctx, b.key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", false, nil
		}
		return "", false, err
	}
	return val, true, nil
}

// Bytes retrieves the raw byte slice without unmarshalling.
func (b *ClusterRunBuilder) Bytes() ([]byte, bool, error) {
	val, err := b.client.rdb.Get(b.ctx, b.key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}
	return val, true, nil
}

func (b *ClusterRunBuilder) unmarshal(data []byte, dest any) error {
	if strPtr, ok := dest.(*string); ok {
		*strPtr = string(data)
		return nil
	}
	if bytesPtr, ok := dest.(*[]byte); ok {
		*bytesPtr = data
		return nil
	}
	return json.Unmarshal(data, dest)
}

// ClusterSprintBuilder provides a fluent API for atomic Redis Cluster operations.
type ClusterSprintBuilder struct {
	ctx    context.Context
	client *ClusterClient
	key    string
}

// Incr increments the value by 1.
func (b *ClusterSprintBuilder) Incr() (int64, error) {
	return b.client.rdb.Incr(b.ctx, b.key).Result()
}

// IncrBy increments the value by the specified amount.
func (b *ClusterSprintBuilder) IncrBy(n int64) (int64, error) {
	return b.client.rdb.IncrBy(b.ctx, b.key, n).Result()
}

// Decr decrements the value by 1.
func (b *ClusterSprintBuilder) Decr() (int64, error) {
	return b.client.rdb.Decr(b.ctx, b.key).Result()
}

// DecrBy decrements the value by the specified amount.
func (b *ClusterSprintBuilder) DecrBy(n int64) (int64, error) {
	return b.client.rdb.DecrBy(b.ctx, b.key, n).Result()
}

// IncrByFloat increments the value by a float amount.
func (b *ClusterSprintBuilder) IncrByFloat(n float64) (float64, error) {
	return b.client.rdb.IncrByFloat(b.ctx, b.key, n).Result()
}

// Get returns the current value as int64.
func (b *ClusterSprintBuilder) Get() (int64, error) {
	val, err := b.client.rdb.Get(b.ctx, b.key).Int64()
	if err != nil {
		return 0, nil
	}
	return val, nil
}

// SetWithTTL sets the counter to a specific value with TTL.
func (b *ClusterSprintBuilder) SetWithTTL(value int64, ttl time.Duration) error {
	return b.client.rdb.Set(b.ctx, b.key, value, ttl).Err()
}

// Expire sets a TTL on an existing counter.
func (b *ClusterSprintBuilder) Expire(ttl time.Duration) error {
	return b.client.rdb.Expire(b.ctx, b.key, ttl).Err()
}
