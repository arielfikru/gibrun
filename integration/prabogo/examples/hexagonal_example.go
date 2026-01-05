// Example: Using Gib.run with PraboGo
// This shows how to integrate Gib.run into a PraboGo hexagonal architecture

package example

import (
	"context"
	"log"
	"time"

	"github.com/arielfikru/gibrun"
)

// Domain Entity
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Port (Interface)
type UserCacheRepository interface {
	Set(ctx context.Context, user *User) error
	Get(ctx context.Context, id string) (*User, error)
	Delete(ctx context.Context, id string) error
}

// Adapter (Implementation)
type userCacheAdapter struct {
	client *gibrun.Client
	ttl    time.Duration
}

func NewUserCacheAdapter(redisAddr string, ttl time.Duration) UserCacheRepository {
	client := gibrun.New(gibrun.Config{
		Addr: redisAddr,
	})
	return &userCacheAdapter{
		client: client,
		ttl:    ttl,
	}
}

func (a *userCacheAdapter) Set(ctx context.Context, user *User) error {
	return a.client.Gib(ctx, "user:"+user.ID).
		Value(user).
		TTL(a.ttl).
		Exec()
}

func (a *userCacheAdapter) Get(ctx context.Context, id string) (*User, error) {
	var user User
	found, err := a.client.Run(ctx, "user:"+id).Bind(&user)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil // Cache miss
	}
	return &user, nil
}

func (a *userCacheAdapter) Delete(ctx context.Context, id string) error {
	return a.client.Del(ctx, "user:"+id)
}

// Domain Service using Cache
type UserService struct {
	cache    UserCacheRepository
	database UserDatabaseRepository // Your database repository
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
	// Try cache first
	user, err := s.cache.Get(ctx, id)
	if err != nil {
		log.Printf("Cache error: %v", err)
	}
	if user != nil {
		return user, nil // Cache hit
	}

	// Cache miss - get from database
	user, err = s.database.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store in cache for next time
	if user != nil {
		_ = s.cache.Set(ctx, user)
	}

	return user, nil
}

// Stub interface for compilation
type UserDatabaseRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
}
