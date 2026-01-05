package gibrun

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// ScanOptions configures the Blusukan scanner behavior.
// "Blusukan" means going out to meet people directly - here we go out
// to scan keys directly, but safely and without blocking.
type ScanOptions struct {
	// Pattern is the key pattern to scan (e.g., "user:*").
	// Default is "*" (all keys).
	Pattern string

	// Count is a hint to Redis about how many keys to return per iteration.
	// This is NOT a limit - Redis may return more or fewer keys.
	// Default is 100.
	Count int64

	// BatchDelay adds a delay between scan iterations to reduce Redis load.
	// Set to 0 for no delay (fastest, but highest load).
	BatchDelay time.Duration

	// Type filters by Redis data type: "string", "list", "set", "zset", "hash", "stream".
	// Leave empty to scan all types.
	Type string
}

// ScanResult represents a single scanned key with optional metadata.
type ScanResult struct {
	Key string
	// Populated only if WithValues is used
	Value []byte
	// Populated only if WithTTL is used
	TTL time.Duration
}

// Scanner provides a safe, non-blocking way to iterate over keys.
// It uses SCAN instead of KEYS to avoid blocking the Redis server.
type Scanner struct {
	ctx     context.Context
	client  *Client
	opts    ScanOptions
	cursor  uint64
	buffer  []string
	bufIdx  int
	done    bool
	err     error
}

// Blusukan starts a safe key scanning operation.
// Named after the Indonesian practice of leaders going directly to the people.
// This scanner goes directly to the keys, without blocking Redis.
//
// Example:
//
//	scanner := app.Blusukan(ctx, gibrun.ScanOptions{
//	    Pattern: "user:*",
//	    Count:   100,
//	})
//	for scanner.Next() {
//	    fmt.Println(scanner.Key())
//	}
//	if err := scanner.Err(); err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) Blusukan(ctx context.Context, opts ScanOptions) *Scanner {
	if opts.Pattern == "" {
		opts.Pattern = "*"
	}
	if opts.Count == 0 {
		opts.Count = 100
	}

	return &Scanner{
		ctx:    ctx,
		client: c,
		opts:   opts,
	}
}

// Next advances to the next key. Returns false when done or on error.
func (s *Scanner) Next() bool {
	if s.done || s.err != nil {
		return false
	}

	// Check context
	select {
	case <-s.ctx.Done():
		s.err = s.ctx.Err()
		return false
	default:
	}

	// Return from buffer if available
	if s.bufIdx < len(s.buffer) {
		s.bufIdx++
		return true
	}

	// Need to fetch more keys
	s.buffer = nil
	s.bufIdx = 0

	// Apply batch delay if configured
	if s.opts.BatchDelay > 0 && s.cursor > 0 {
		time.Sleep(s.opts.BatchDelay)
	}

	// Scan based on type filter
	var keys []string
	var cursor uint64
	var err error

	if s.opts.Type != "" {
		// Use SCAN with TYPE filter (Redis 6.0+)
		keys, cursor, err = s.client.rdb.ScanType(s.ctx, s.cursor, s.opts.Pattern, s.opts.Count, s.opts.Type).Result()
	} else {
		keys, cursor, err = s.client.rdb.Scan(s.ctx, s.cursor, s.opts.Pattern, s.opts.Count).Result()
	}

	if err != nil {
		s.err = err
		return false
	}

	s.cursor = cursor
	s.buffer = keys

	// Check if we're done
	if cursor == 0 {
		s.done = true
	}

	// Return first key if we have any
	if len(s.buffer) > 0 {
		s.bufIdx = 1
		return true
	}

	// Buffer empty but not done - recurse to get more
	if !s.done {
		return s.Next()
	}

	return false
}

// Key returns the current key. Call after Next() returns true.
func (s *Scanner) Key() string {
	if s.bufIdx > 0 && s.bufIdx <= len(s.buffer) {
		return s.buffer[s.bufIdx-1]
	}
	return ""
}

// Err returns any error that occurred during scanning.
func (s *Scanner) Err() error {
	return s.err
}

// All collects all matching keys into a slice.
// Use with caution on large datasets - prefer iterating with Next().
//
// Example:
//
//	keys, err := app.Blusukan(ctx, gibrun.ScanOptions{Pattern: "session:*"}).All()
func (s *Scanner) All() ([]string, error) {
	var keys []string
	for s.Next() {
		keys = append(keys, s.Key())
	}
	return keys, s.Err()
}

// Each calls the provided function for each matching key.
// Return false from the function to stop iteration early.
//
// Example:
//
//	app.Blusukan(ctx, opts).Each(func(key string) bool {
//	    fmt.Println(key)
//	    return true // continue
//	})
func (s *Scanner) Each(fn func(key string) bool) error {
	for s.Next() {
		if !fn(s.Key()) {
			break
		}
	}
	return s.Err()
}

// Count returns the total number of matching keys.
// This scans through all keys, so it may take time for large datasets.
func (s *Scanner) Count() (int, error) {
	count := 0
	for s.Next() {
		count++
	}
	return count, s.Err()
}

// ClusterScanner provides safe scanning across Redis Cluster shards.
type ClusterScanner struct {
	ctx     context.Context
	client  *ClusterClient
	opts    ScanOptions
	masters []*redis.Client
	current int
	scanner *redis.ScanIterator
}

// Blusukan starts a safe key scanning operation on the cluster.
// Scans across all master nodes to cover all shards.
func (c *ClusterClient) Blusukan(ctx context.Context, opts ScanOptions) *ClusterScanner {
	if opts.Pattern == "" {
		opts.Pattern = "*"
	}
	if opts.Count == 0 {
		opts.Count = 100
	}

	return &ClusterScanner{
		ctx:    ctx,
		client: c,
		opts:   opts,
	}
}

// All collects all matching keys from all cluster shards.
func (s *ClusterScanner) All() ([]string, error) {
	var allKeys []string

	err := s.client.rdb.ForEachMaster(s.ctx, func(ctx context.Context, master *redis.Client) error {
		var cursor uint64
		for {
			// Apply delay between batches
			if s.opts.BatchDelay > 0 && cursor > 0 {
				time.Sleep(s.opts.BatchDelay)
			}

			var keys []string
			var err error
			if s.opts.Type != "" {
				keys, cursor, err = master.ScanType(ctx, cursor, s.opts.Pattern, s.opts.Count, s.opts.Type).Result()
			} else {
				keys, cursor, err = master.Scan(ctx, cursor, s.opts.Pattern, s.opts.Count).Result()
			}
			if err != nil {
				return err
			}
			allKeys = append(allKeys, keys...)
			if cursor == 0 {
				break
			}
		}
		return nil
	})

	return allKeys, err
}

// Each calls the provided function for each matching key across all shards.
func (s *ClusterScanner) Each(fn func(key string) bool) error {
	stopped := false

	err := s.client.rdb.ForEachMaster(s.ctx, func(ctx context.Context, master *redis.Client) error {
		if stopped {
			return nil
		}

		var cursor uint64
		for {
			if s.opts.BatchDelay > 0 && cursor > 0 {
				time.Sleep(s.opts.BatchDelay)
			}

			var keys []string
			var err error
			if s.opts.Type != "" {
				keys, cursor, err = master.ScanType(ctx, cursor, s.opts.Pattern, s.opts.Count, s.opts.Type).Result()
			} else {
				keys, cursor, err = master.Scan(ctx, cursor, s.opts.Pattern, s.opts.Count).Result()
			}
			if err != nil {
				return err
			}

			for _, key := range keys {
				if !fn(key) {
					stopped = true
					return nil
				}
			}

			if cursor == 0 {
				break
			}
		}
		return nil
	})

	return err
}

// Count returns the total number of matching keys across all shards.
func (s *ClusterScanner) Count() (int, error) {
	count := 0
	err := s.Each(func(key string) bool {
		count++
		return true
	})
	return count, err
}
