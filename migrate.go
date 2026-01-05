package gibrun

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// MigrateOptions configures the migration behavior.
type MigrateOptions struct {
	// Pattern is the key pattern to migrate (e.g., "user:*").
	// If empty, all keys will be migrated.
	Pattern string

	// BatchSize is the number of keys to process in each batch.
	// Default is 100.
	BatchSize int

	// TTL sets a new TTL for migrated keys.
	// If zero, the original TTL is preserved.
	TTL time.Duration

	// PreserveTTL attempts to preserve original TTL from source.
	// If false and TTL is zero, keys will have no expiration.
	PreserveTTL bool

	// OnProgress is called after each batch is processed.
	// done is the number of keys migrated so far.
	// total is the estimated total (-1 if unknown).
	OnProgress func(done, total int)

	// OnError is called when a key fails to migrate.
	// Return true to continue, false to abort.
	OnError func(key string, err error) bool

	// DryRun if true, only counts keys without actually migrating.
	DryRun bool
}

// MigrateResult contains the result of a migration operation.
type MigrateResult struct {
	// TotalKeys is the number of keys found matching the pattern.
	TotalKeys int

	// MigratedKeys is the number of keys successfully migrated.
	MigratedKeys int

	// FailedKeys is the number of keys that failed to migrate.
	FailedKeys int

	// Duration is how long the migration took.
	Duration time.Duration

	// Errors contains individual key errors if any.
	Errors []MigrateError
}

// MigrateError represents a single key migration failure.
type MigrateError struct {
	Key   string
	Error error
}

// Migrate transfers data from source to destination Redis.
// This is the "hilirisasi" of data - moving raw resources (keys)
// from one region to another for localized processing.
//
// Example:
//
//	result, err := gibrun.Migrate(ctx, srcClient, dstClient, gibrun.MigrateOptions{
//	    Pattern:   "user:*",
//	    BatchSize: 100,
//	    OnProgress: func(done, total int) {
//	        fmt.Printf("Progress: %d/%d\n", done, total)
//	    },
//	})
func Migrate(ctx context.Context, src, dst *Client, opts MigrateOptions) (*MigrateResult, error) {
	startTime := time.Now()

	// Set defaults
	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}
	if opts.Pattern == "" {
		opts.Pattern = "*"
	}

	result := &MigrateResult{}

	// Scan all keys matching pattern
	keys, err := scanAllKeys(ctx, src, opts.Pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan keys: %w", err)
	}

	result.TotalKeys = len(keys)

	if opts.DryRun {
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Process in batches
	for i := 0; i < len(keys); i += opts.BatchSize {
		select {
		case <-ctx.Done():
			result.Duration = time.Since(startTime)
			return result, ctx.Err()
		default:
		}

		end := i + opts.BatchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		// Migrate batch
		for _, key := range batch {
			if err := migrateKey(ctx, src, dst, key, opts); err != nil {
				result.FailedKeys++
				result.Errors = append(result.Errors, MigrateError{Key: key, Error: err})

				if opts.OnError != nil {
					if !opts.OnError(key, err) {
						result.Duration = time.Since(startTime)
						return result, fmt.Errorf("migration aborted at key %s: %w", key, err)
					}
				}
			} else {
				result.MigratedKeys++
			}
		}

		// Report progress
		if opts.OnProgress != nil {
			opts.OnProgress(result.MigratedKeys+result.FailedKeys, result.TotalKeys)
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// MigrateCluster transfers data from a cluster source to destination.
// Works across Redis Cluster shards.
func MigrateCluster(ctx context.Context, src *ClusterClient, dst *Client, opts MigrateOptions) (*MigrateResult, error) {
	startTime := time.Now()

	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}
	if opts.Pattern == "" {
		opts.Pattern = "*"
	}

	result := &MigrateResult{}

	// Scan all keys from cluster
	keys, err := scanAllClusterKeys(ctx, src, opts.Pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan cluster keys: %w", err)
	}

	result.TotalKeys = len(keys)

	if opts.DryRun {
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Process in batches
	for i := 0; i < len(keys); i += opts.BatchSize {
		select {
		case <-ctx.Done():
			result.Duration = time.Since(startTime)
			return result, ctx.Err()
		default:
		}

		end := i + opts.BatchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		for _, key := range batch {
			if err := migrateClusterKey(ctx, src, dst, key, opts); err != nil {
				result.FailedKeys++
				result.Errors = append(result.Errors, MigrateError{Key: key, Error: err})

				if opts.OnError != nil {
					if !opts.OnError(key, err) {
						result.Duration = time.Since(startTime)
						return result, fmt.Errorf("migration aborted at key %s: %w", key, err)
					}
				}
			} else {
				result.MigratedKeys++
			}
		}

		if opts.OnProgress != nil {
			opts.OnProgress(result.MigratedKeys+result.FailedKeys, result.TotalKeys)
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// scanAllKeys scans all keys matching the pattern from a single-node client.
func scanAllKeys(ctx context.Context, client *Client, pattern string) ([]string, error) {
	var keys []string
	var cursor uint64

	for {
		var batch []string
		var err error
		batch, cursor, err = client.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// scanAllClusterKeys scans all keys from all shards in a cluster.
func scanAllClusterKeys(ctx context.Context, client *ClusterClient, pattern string) ([]string, error) {
	var allKeys []string

	err := client.rdb.ForEachMaster(ctx, func(ctx context.Context, master *redis.Client) error {
		var cursor uint64
		for {
			batch, newCursor, err := master.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return err
			}
			allKeys = append(allKeys, batch...)
			cursor = newCursor
			if cursor == 0 {
				break
			}
		}
		return nil
	})

	return allKeys, err
}

// migrateKey migrates a single key from src to dst.
func migrateKey(ctx context.Context, src, dst *Client, key string, opts MigrateOptions) error {
	// Get value
	data, err := src.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}

	// Get TTL if preserving
	var ttl time.Duration
	if opts.PreserveTTL {
		ttl, err = src.rdb.TTL(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("ttl failed: %w", err)
		}
		// TTL returns -1 for no expiration, -2 for key not found
		if ttl < 0 {
			ttl = 0
		}
	} else if opts.TTL > 0 {
		ttl = opts.TTL
	}

	// Set value
	if err := dst.rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("set failed: %w", err)
	}

	return nil
}

// migrateClusterKey migrates a single key from cluster src to dst.
func migrateClusterKey(ctx context.Context, src *ClusterClient, dst *Client, key string, opts MigrateOptions) error {
	// Get value from cluster
	data, err := src.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}

	// Get TTL if preserving
	var ttl time.Duration
	if opts.PreserveTTL {
		ttl, err = src.rdb.TTL(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("ttl failed: %w", err)
		}
		if ttl < 0 {
			ttl = 0
		}
	} else if opts.TTL > 0 {
		ttl = opts.TTL
	}

	// Set value to single-node destination
	if err := dst.rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("set failed: %w", err)
	}

	return nil
}
