# Gib.run

> **"Give Data. Run Fast."**

The opinionated Redis framework for Go developers who prioritize acceleration, sustainability, and high-performance.

## Installation

```bash
go get github.com/arielfikru/gibrun
```

## Quick Start

```go
package main

import (
    "context"
    "time"
    "github.com/arielfikru/gibrun"
)

func main() {
    // Connect to Redis
    app := gibrun.New(gibrun.Config{
        Addr: "localhost:6379",
    })
    defer app.Close()

    ctx := context.Background()

    // Store data (Gib)
    type User struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }

    user := User{Name: "John", Email: "john@example.com"}
    app.Gib(ctx, "user:123").Value(user).TTL(5 * time.Minute).Exec()

    // Retrieve data (Run)
    var result User
    found, _ := app.Run(ctx, "user:123").Bind(&result)
    if found {
        fmt.Printf("User: %s\n", result.Name)
    }

    // Atomic operations (Sprint)
    count, _ := app.Sprint(ctx, "counter:visitors").Incr()
    fmt.Printf("Visitors: %d\n", count)
}
```

## Features

### Core Operations

- **Gib** - Store data with automatic JSON marshalling
- **Run** - Retrieve data with automatic unmarshalling  
- **Sprint** - Atomic counter operations

### Redis Cluster

```go
cluster := gibrun.NewCluster(gibrun.ClusterConfig{
    Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
})

// Same API as single-node
cluster.Gib(ctx, "key").Value(data).Exec()
```

### Auto-Migration

Transfer data between Redis instances:

```go
result, _ := gibrun.Migrate(ctx, srcClient, dstClient, gibrun.MigrateOptions{
    Pattern:   "user:*",
    BatchSize: 100,
    OnProgress: func(done, total int) {
        fmt.Printf("Progress: %d/%d\n", done, total)
    },
})
```

### Blusukan Scanner

Safe key scanning without blocking Redis:

```go
scanner := app.Blusukan(ctx, gibrun.ScanOptions{
    Pattern: "session:*",
    Count:   100,
})

for scanner.Next() {
    fmt.Println(scanner.Key())
}
```

### Rate Limiting

Token bucket rate limiter with HTTP middleware:

```go
limiter := gibrun.NewRateLimiter(client, gibrun.RateLimitConfig{
    Rate:   100,           // 100 requests
    Window: time.Minute,   // per minute
})

// Use as middleware
http.Handle("/api/", limiter.Middleware(apiHandler))

// Or check manually
result, _ := limiter.Allow(ctx, "user:123")
if !result.Allowed {
    // Rate limited
}
```

## API Reference

### Client Methods

| Method | Description |
|--------|-------------|
| `New(Config)` | Create single-node client |
| `NewCluster(ClusterConfig)` | Create cluster client |
| `Gib(ctx, key)` | Start store operation |
| `Run(ctx, key)` | Start retrieve operation |
| `Sprint(ctx, key)` | Start atomic operation |
| `Blusukan(ctx, opts)` | Start key scanner |
| `Del(ctx, keys...)` | Delete keys |
| `Exists(ctx, key)` | Check if key exists |

### Gib Builder

| Method | Description |
|--------|-------------|
| `.Value(v)` | Set value to store |
| `.TTL(d)` | Set expiration duration |
| `.Exec()` | Execute store operation |

### Run Builder

| Method | Description |
|--------|-------------|
| `.Bind(&v)` | Unmarshal to pointer |
| `.Raw()` | Get raw string |
| `.Bytes()` | Get raw bytes |

### Sprint Builder

| Method | Description |
|--------|-------------|
| `.Incr()` | Increment by 1 |
| `.IncrBy(n)` | Increment by n |
| `.Decr()` | Decrement by 1 |
| `.DecrBy(n)` | Decrement by n |

## Documentation

Visit [https://arielfikru.github.io/gibrun](https://arielfikru.github.io/gibrun) for full documentation.

## License

MIT License
