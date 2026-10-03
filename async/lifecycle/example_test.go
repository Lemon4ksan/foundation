package lifecycle_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lemon4ksan/foundation/async/lifecycle"
)

type Database struct{}

func (d *Database) Name() string           { return "postgres" }
func (d *Database) Dependencies() []string { return nil }
func (d *Database) Init(ctx context.Context) error {
	fmt.Println("1. Postgres initialized")
	return nil
}

func (d *Database) Start(ctx context.Context) error { fmt.Println("1. Postgres connected"); return nil }

func (d *Database) Stop(ctx context.Context) error { fmt.Println("Postgres disconnected"); return nil }

type RedisCache struct{}

func (r *RedisCache) Name() string                   { return "redis" }
func (r *RedisCache) Dependencies() []string         { return []string{"postgres"} }
func (r *RedisCache) Init(ctx context.Context) error { fmt.Println("2. Redis initialized"); return nil }

func (r *RedisCache) Start(ctx context.Context) error { fmt.Println("2. Redis connected"); return nil }

func (r *RedisCache) Stop(ctx context.Context) error { fmt.Println("Redis disconnected"); return nil }

type HTTPServer struct{}

func (s *HTTPServer) Name() string           { return "http-server" }
func (s *HTTPServer) Dependencies() []string { return []string{"postgres", "redis"} }
func (s *HTTPServer) Init(ctx context.Context) error {
	fmt.Println("3. HTTP routes registered")
	return nil
}

func (s *HTTPServer) Start(ctx context.Context) error { fmt.Println("3. HTTP listening"); return nil }

func (s *HTTPServer) Stop(ctx context.Context) error { fmt.Println("HTTP server drained"); return nil }

func ExampleOrchestrator() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	orch := lifecycle.NewOrchestrator()
	orch.Register(&HTTPServer{})
	orch.Register(&RedisCache{})
	orch.Register(&Database{})

	if err := orch.InitAll(ctx); err != nil {
		log.Fatalf("Init failed: %v", err)
	}

	if err := orch.StartAll(ctx); err != nil {
		log.Fatalf("Start failed: %v", err)
	}

	_ = orch.StopAll(context.Background())

	// Output:
	// 1. Postgres initialized
	// 2. Redis initialized
	// 3. HTTP routes registered
	// 1. Postgres connected
	// 2. Redis connected
	// 3. HTTP listening
	// HTTP server drained
	// Redis disconnected
	// Postgres disconnected
}
