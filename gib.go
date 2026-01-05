package gibrun

import (
	"context"
	"encoding/json"
	"time"
)

// GibBuilder provides a fluent API for storing data in Redis.
// It handles automatic JSON marshalling for struct types.
type GibBuilder struct {
	ctx    context.Context
	client *Client
	key    string
	value  any
	ttl    time.Duration
}

// Value sets the data to be stored.
// Structs will be automatically marshalled to JSON.
// Primitive types (string, int, etc.) will be stored directly.
//
// Example:
//
//	app.Gib(ctx, "user:123").Value(userStruct)
func (b *GibBuilder) Value(v any) *GibBuilder {
	b.value = v
	return b
}

// TTL sets the time-to-live for the cached data.
// If not called, the data will persist indefinitely.
//
// Example:
//
//	app.Gib(ctx, "session").Value(data).TTL(30 * time.Minute)
func (b *GibBuilder) TTL(d time.Duration) *GibBuilder {
	b.ttl = d
	return b
}

// Exec executes the storage operation.
// This is where the "downstreaming" happens - raw data gets transformed
// and stored in Redis.
//
// Example:
//
//	err := app.Gib(ctx, "key").Value(data).TTL(5*time.Minute).Exec()
func (b *GibBuilder) Exec() error {
	if b.value == nil {
		return ErrNilValue
	}

	// Auto-downstreaming: marshal struct to JSON
	data, err := b.marshal(b.value)
	if err != nil {
		return err
	}

	// Store in Redis with optional TTL
	if b.ttl > 0 {
		return b.client.rdb.Set(b.ctx, b.key, data, b.ttl).Err()
	}
	return b.client.rdb.Set(b.ctx, b.key, data, 0).Err()
}

// marshal converts the value to a storable format.
// Supports automatic JSON marshalling for complex types.
func (b *GibBuilder) marshal(v any) ([]byte, error) {
	switch val := v.(type) {
	case string:
		return []byte(val), nil
	case []byte:
		return val, nil
	default:
		// Auto-downstreaming: marshal struct/slice/map to JSON
		return json.Marshal(val)
	}
}
