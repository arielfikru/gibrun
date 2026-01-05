# Gib.run + PraboGo Integration

This directory contains integration templates and examples for using Gib.run with PraboGo framework.

## Quick Setup

1. Add Gib.run to your PraboGo project:

```bash
go get github.com/arielfikru/gibrun@v1.0.0
```

2. Copy the templates to your PraboGo project's template directory.

## Templates Included

| Template | Description |
|----------|-------------|
| `cache_adapter.go.tmpl` | Redis cache adapter using Gib.run |
| `cache_repository.go.tmpl` | Repository interface for caching |
| `ratelimit_middleware.go.tmpl` | Rate limiting middleware for Fiber |

## Usage Example

After copying templates, generate a cache adapter:

```bash
make outbound-cache-gibrun VAL=user
```

This will generate:
- `internal/adapter/outbound/cache/user/adapter.go`
- `internal/adapter/outbound/cache/user/repository.go`
