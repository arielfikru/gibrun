package gibrun

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// ClusterConfig holds the configuration for connecting to a Redis Cluster.
// The Capital Update - connecting multiple nodes across regions.
type ClusterConfig struct {
	// Addrs is a list of Redis Cluster node addresses.
	// Example: []string{"node1:6379", "node2:6379", "node3:6379"}
	Addrs []string

	// Password for Redis authentication (same for all nodes).
	Password string

	// MaxRedirects is the maximum number of redirects before giving up.
	// Default is 3.
	MaxRedirects int

	// ReadOnly enables read-only mode for replica nodes.
	// Distributes read load across replicas.
	ReadOnly bool

	// RouteByLatency routes commands to the node with lowest latency.
	RouteByLatency bool

	// RouteRandomly routes commands randomly across nodes.
	RouteRandomly bool
}

// ClusterClient is the gibrun client for Redis Cluster mode.
// Provides the same Gib/Run/Sprint API as the single-node Client.
type ClusterClient struct {
	rdb *redis.ClusterClient
}

// NewCluster creates a new gibrun ClusterClient for Redis Cluster mode.
// This enables horizontal scaling and high availability across multiple nodes.
//
// Example:
//
//	cluster := gibrun.NewCluster(gibrun.ClusterConfig{
//	    Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
//	})
func NewCluster(cfg ClusterConfig) *ClusterClient {
	maxRedirects := cfg.MaxRedirects
	if maxRedirects == 0 {
		maxRedirects = 3
	}

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:          cfg.Addrs,
		Password:       cfg.Password,
		MaxRedirects:   maxRedirects,
		ReadOnly:       cfg.ReadOnly,
		RouteByLatency: cfg.RouteByLatency,
		RouteRandomly:  cfg.RouteRandomly,
	})

	return &ClusterClient{
		rdb: rdb,
	}
}

// Ping checks the connection to the Redis Cluster.
func (c *ClusterClient) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Close closes the Redis Cluster connection.
func (c *ClusterClient) Close() error {
	return c.rdb.Close()
}

// Gib starts a data storage operation on the cluster.
//
// Example:
//
//	err := cluster.Gib(ctx, "key").Value(data).TTL(5*time.Minute).Exec()
func (c *ClusterClient) Gib(ctx context.Context, key string) *ClusterGibBuilder {
	return &ClusterGibBuilder{
		ctx:    ctx,
		client: c,
		key:    key,
	}
}

// Run starts a data retrieval operation on the cluster.
//
// Example:
//
//	found, err := cluster.Run(ctx, "key").Bind(&result)
func (c *ClusterClient) Run(ctx context.Context, key string) *ClusterRunBuilder {
	return &ClusterRunBuilder{
		ctx:    ctx,
		client: c,
		key:    key,
	}
}

// Sprint starts an atomic operation on the cluster.
//
// Example:
//
//	count, _ := cluster.Sprint(ctx, "counter").Incr()
func (c *ClusterClient) Sprint(ctx context.Context, key string) *ClusterSprintBuilder {
	return &ClusterSprintBuilder{
		ctx:    ctx,
		client: c,
		key:    key,
	}
}

// Del deletes one or more keys from the cluster.
func (c *ClusterClient) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in the cluster.
func (c *ClusterClient) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ForEachShard executes a function on each shard/master node.
// Useful for operations that need to touch all nodes.
func (c *ClusterClient) ForEachShard(ctx context.Context, fn func(ctx context.Context, client *redis.Client) error) error {
	return c.rdb.ForEachShard(ctx, fn)
}

// ClusterSlots returns information about the cluster slot distribution.
func (c *ClusterClient) ClusterSlots(ctx context.Context) ([]redis.ClusterSlot, error) {
	return c.rdb.ClusterSlots(ctx).Result()
}
