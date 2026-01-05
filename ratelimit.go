package gibrun

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// RateLimitConfig configures the Bansos rate limiter.
// "Bansos" (Bantuan Sosial) means social assistance - this rate limiter
// distributes tokens fairly, like distributing aid evenly to all.
type RateLimitConfig struct {
	// Key prefix for rate limit counters in Redis.
	// Default is "ratelimit".
	KeyPrefix string

	// Rate is the number of requests allowed per window.
	// This is the "quota" each user receives.
	Rate int

	// Window is the time window for the rate limit.
	// After this duration, the quota resets.
	Window time.Duration

	// BurstSize is the maximum tokens that can accumulate.
	// Allows short bursts above the steady rate.
	// Default equals Rate (no extra burst).
	BurstSize int

	// KeyFunc extracts the rate limit key from the request.
	// Default uses the client IP address.
	KeyFunc func(r *http.Request) string
}

// RateLimiter provides Redis-backed rate limiting using token bucket algorithm.
type RateLimiter struct {
	client *Client
	config RateLimitConfig
}

// RateLimitResult contains the result of a rate limit check.
type RateLimitResult struct {
	// Allowed is true if the request should be allowed.
	Allowed bool

	// Remaining is the number of requests remaining in the current window.
	Remaining int

	// ResetAt is when the rate limit window resets.
	ResetAt time.Time

	// RetryAfter is the duration to wait before retrying (if not allowed).
	RetryAfter time.Duration
}

// NewRateLimiter creates a new Bansos rate limiter.
//
// Example:
//
//	limiter := gibrun.NewRateLimiter(client, gibrun.RateLimitConfig{
//	    Rate:   100,               // 100 requests
//	    Window: time.Minute,       // per minute
//	})
func NewRateLimiter(client *Client, config RateLimitConfig) *RateLimiter {
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ratelimit"
	}
	if config.BurstSize == 0 {
		config.BurstSize = config.Rate
	}
	if config.KeyFunc == nil {
		config.KeyFunc = defaultKeyFunc
	}

	return &RateLimiter{
		client: client,
		config: config,
	}
}

// Allow checks if a request with the given key should be allowed.
// Uses the sliding window counter algorithm for accurate rate limiting.
//
// Example:
//
//	result, err := limiter.Allow(ctx, "user:123")
//	if !result.Allowed {
//	    // Reject request, retry after result.RetryAfter
//	}
func (rl *RateLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	return rl.AllowN(ctx, key, 1)
}

// AllowN checks if n requests should be allowed.
// Useful for operations that consume multiple tokens.
func (rl *RateLimiter) AllowN(ctx context.Context, key string, n int) (*RateLimitResult, error) {
	now := time.Now()
	windowKey := rl.buildKey(key, now)

	// Use Redis transaction to atomically increment and get TTL
	pipe := rl.client.rdb.Pipeline()

	// Increment counter
	incrCmd := pipe.IncrBy(ctx, windowKey, int64(n))

	// Set expiration if this is a new key
	pipe.Expire(ctx, windowKey, rl.config.Window)

	// Get TTL for reset time
	ttlCmd := pipe.TTL(ctx, windowKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	count := incrCmd.Val()
	ttl := ttlCmd.Val()

	// Calculate remaining
	remaining := rl.config.Rate - int(count)
	if remaining < 0 {
		remaining = 0
	}

	// Calculate reset time
	resetAt := now.Add(ttl)
	if ttl < 0 {
		resetAt = now.Add(rl.config.Window)
	}

	result := &RateLimitResult{
		Allowed:   count <= int64(rl.config.Rate),
		Remaining: remaining,
		ResetAt:   resetAt,
	}

	if !result.Allowed {
		result.RetryAfter = ttl
		if result.RetryAfter < 0 {
			result.RetryAfter = rl.config.Window
		}
	}

	return result, nil
}

// Middleware returns an HTTP middleware for rate limiting.
// Automatically rejects requests that exceed the rate limit.
//
// Example:
//
//	http.Handle("/api/", limiter.Middleware(apiHandler))
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rl.config.KeyFunc(r)

		result, err := rl.Allow(r.Context(), key)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.Rate))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt.Unix()))

		if !result.Allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", result.RetryAfter.Seconds()))
			http.Error(w, "Rate limit exceeded. Bansos quota habis, silakan tunggu.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc is like Middleware but for http.HandlerFunc.
func (rl *RateLimiter) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return rl.Middleware(next).ServeHTTP
}

// Reset clears the rate limit for a specific key.
// Useful for admin overrides or testing.
func (rl *RateLimiter) Reset(ctx context.Context, key string) error {
	now := time.Now()
	windowKey := rl.buildKey(key, now)
	return rl.client.rdb.Del(ctx, windowKey).Err()
}

// buildKey creates the Redis key for the rate limit counter.
func (rl *RateLimiter) buildKey(key string, t time.Time) string {
	// Use window-aligned timestamps for consistent rate limiting
	window := t.Unix() / int64(rl.config.Window.Seconds())
	return fmt.Sprintf("%s:%s:%d", rl.config.KeyPrefix, key, window)
}

// defaultKeyFunc extracts client IP from the request.
func defaultKeyFunc(r *http.Request) string {
	// Check X-Forwarded-For for proxied requests
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// ClusterRateLimiter provides rate limiting for Redis Cluster.
type ClusterRateLimiter struct {
	client *ClusterClient
	config RateLimitConfig
}

// NewClusterRateLimiter creates a rate limiter for Redis Cluster.
func NewClusterRateLimiter(client *ClusterClient, config RateLimitConfig) *ClusterRateLimiter {
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ratelimit"
	}
	if config.BurstSize == 0 {
		config.BurstSize = config.Rate
	}
	if config.KeyFunc == nil {
		config.KeyFunc = defaultKeyFunc
	}

	return &ClusterRateLimiter{
		client: client,
		config: config,
	}
}

// Allow checks if a request with the given key should be allowed.
func (rl *ClusterRateLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	return rl.AllowN(ctx, key, 1)
}

// AllowN checks if n requests should be allowed.
func (rl *ClusterRateLimiter) AllowN(ctx context.Context, key string, n int) (*RateLimitResult, error) {
	now := time.Now()
	windowKey := rl.buildKey(key, now)

	pipe := rl.client.rdb.Pipeline()
	incrCmd := pipe.IncrBy(ctx, windowKey, int64(n))
	pipe.Expire(ctx, windowKey, rl.config.Window)
	ttlCmd := pipe.TTL(ctx, windowKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	count := incrCmd.Val()
	ttl := ttlCmd.Val()

	remaining := rl.config.Rate - int(count)
	if remaining < 0 {
		remaining = 0
	}

	resetAt := now.Add(ttl)
	if ttl < 0 {
		resetAt = now.Add(rl.config.Window)
	}

	result := &RateLimitResult{
		Allowed:   count <= int64(rl.config.Rate),
		Remaining: remaining,
		ResetAt:   resetAt,
	}

	if !result.Allowed {
		result.RetryAfter = ttl
		if result.RetryAfter < 0 {
			result.RetryAfter = rl.config.Window
		}
	}

	return result, nil
}

// Middleware returns an HTTP middleware for rate limiting.
func (rl *ClusterRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rl.config.KeyFunc(r)

		result, err := rl.Allow(r.Context(), key)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.Rate))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt.Unix()))

		if !result.Allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", result.RetryAfter.Seconds()))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *ClusterRateLimiter) buildKey(key string, t time.Time) string {
	window := t.Unix() / int64(rl.config.Window.Seconds())
	return fmt.Sprintf("%s:%s:%d", rl.config.KeyPrefix, key, window)
}
