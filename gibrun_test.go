package gibrun_test

import (
	"context"
	"testing"
	"time"

	"github.com/arielfikru/gibrun"
)

// TestConfig tests that configuration is properly applied
func TestConfig(t *testing.T) {
	cfg := gibrun.Config{
		Addr:     "localhost:6379",
		Password: "secret",
		DB:       5,
	}

	if cfg.Addr != "localhost:6379" {
		t.Errorf("expected Addr to be localhost:6379, got %s", cfg.Addr)
	}
	if cfg.Password != "secret" {
		t.Errorf("expected Password to be secret, got %s", cfg.Password)
	}
	if cfg.DB != 5 {
		t.Errorf("expected DB to be 5, got %d", cfg.DB)
	}
}

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	client := gibrun.New(gibrun.Config{
		Addr: "localhost:6379",
	})

	if client == nil {
		t.Error("expected client to not be nil")
	}
}

// Integration tests (require Redis to be running)
// Run with: go test -tags=integration

type TestStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestGibAndRun(t *testing.T) {
	client := gibrun.New(gibrun.Config{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Skip if Redis is not available
	if err := client.Ping(ctx); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	// Test storing and retrieving a struct
	key := "test:gibrun:struct"
	original := TestStruct{Name: "test", Value: 42}

	// Gib (store)
	err := client.Gib(ctx, key).Value(original).TTL(time.Minute).Exec()
	if err != nil {
		t.Fatalf("Gib failed: %v", err)
	}

	// Run (retrieve)
	var result TestStruct
	found, err := client.Run(ctx, key).Bind(&result)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !found {
		t.Fatal("expected to find data")
	}
	if result.Name != original.Name || result.Value != original.Value {
		t.Errorf("expected %+v, got %+v", original, result)
	}

	// Cleanup
	client.Del(ctx, key)
}

func TestGibString(t *testing.T) {
	client := gibrun.New(gibrun.Config{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	key := "test:gibrun:string"
	original := "hello gibrun"

	err := client.Gib(ctx, key).Value(original).Exec()
	if err != nil {
		t.Fatalf("Gib string failed: %v", err)
	}

	val, found, err := client.Run(ctx, key).Raw()
	if err != nil {
		t.Fatalf("Run Raw failed: %v", err)
	}
	if !found {
		t.Fatal("expected to find data")
	}
	if val != original {
		t.Errorf("expected %s, got %s", original, val)
	}

	client.Del(ctx, key)
}

func TestSprint(t *testing.T) {
	client := gibrun.New(gibrun.Config{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	key := "test:gibrun:counter"

	// Clean start
	client.Del(ctx, key)

	// Incr
	val, err := client.Sprint(ctx, key).Incr()
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if val != 1 {
		t.Errorf("expected 1, got %d", val)
	}

	// IncrBy
	val, err = client.Sprint(ctx, key).IncrBy(10)
	if err != nil {
		t.Fatalf("IncrBy failed: %v", err)
	}
	if val != 11 {
		t.Errorf("expected 11, got %d", val)
	}

	// Decr
	val, err = client.Sprint(ctx, key).Decr()
	if err != nil {
		t.Fatalf("Decr failed: %v", err)
	}
	if val != 10 {
		t.Errorf("expected 10, got %d", val)
	}

	client.Del(ctx, key)
}

func TestRunNotFound(t *testing.T) {
	client := gibrun.New(gibrun.Config{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	var result TestStruct
	found, err := client.Run(ctx, "nonexistent:key:12345").Bind(&result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected not found")
	}
}

func TestGibNilValue(t *testing.T) {
	client := gibrun.New(gibrun.Config{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	err := client.Gib(ctx, "test:nil").Exec()
	if err != gibrun.ErrNilValue {
		t.Errorf("expected ErrNilValue, got %v", err)
	}
}

func TestExists(t *testing.T) {
	client := gibrun.New(gibrun.Config{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	key := "test:gibrun:exists"

	// Should not exist
	exists, err := client.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("expected key to not exist")
	}

	// Create key
	client.Gib(ctx, key).Value("test").Exec()

	// Should exist now
	exists, err = client.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("expected key to exist")
	}

	client.Del(ctx, key)
}
