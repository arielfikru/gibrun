package gibrun

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

// RunBuilder provides a fluent API for retrieving data from Redis.
// It handles automatic JSON unmarshalling to target types.
type RunBuilder struct {
	ctx    context.Context
	client *Client
	key    string
}

// Bind retrieves the data and unmarshals it into the provided pointer.
// Returns (true, nil) if data was found and successfully bound.
// Returns (false, nil) if the key doesn't exist (cache miss).
// Returns (false, error) if an error occurred.
//
// Example:
//
//	var user User
//	found, err := app.Run(ctx, "user:123").Bind(&user)
//	if err != nil {
//	    // handle error
//	}
//	if found {
//	    // use user
//	}
func (b *RunBuilder) Bind(dest any) (bool, error) {
	if dest == nil {
		return false, ErrNilPointer
	}

	// Get from Redis
	data, err := b.client.rdb.Get(b.ctx, b.key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Cache miss - data tidak ditemukan, mohon klarifikasi
			return false, nil
		}
		return false, err
	}

	// Unmarshal based on destination type
	if err := b.unmarshal(data, dest); err != nil {
		return false, err
	}

	return true, nil
}

// Raw retrieves the raw string value without unmarshalling.
// Returns (value, true, nil) if found, ("", false, nil) if not found.
//
// Example:
//
//	value, found, err := app.Run(ctx, "simple:key").Raw()
func (b *RunBuilder) Raw() (string, bool, error) {
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
// Returns (value, true, nil) if found, (nil, false, nil) if not found.
func (b *RunBuilder) Bytes() ([]byte, bool, error) {
	val, err := b.client.rdb.Get(b.ctx, b.key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}
	return val, true, nil
}

// unmarshal converts stored data back to the target type.
func (b *RunBuilder) unmarshal(data []byte, dest any) error {
	// Handle string destination directly
	if strPtr, ok := dest.(*string); ok {
		*strPtr = string(data)
		return nil
	}

	// Handle []byte destination directly
	if bytesPtr, ok := dest.(*[]byte); ok {
		*bytesPtr = data
		return nil
	}

	// Default: JSON unmarshal for structs/slices/maps
	return json.Unmarshal(data, dest)
}
