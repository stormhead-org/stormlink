package testcontainers

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestContainers holds all test container instances
type TestContainers struct {
	PostgresContainer *postgres.PostgresContainer
	RedisContainer    *redis.RedisContainer
	PostgresDSN       string
	RedisURL          string
}

// Setup creates and starts all test containers
func Setup(ctx context.Context) (*TestContainers, error) {
	containers := &TestContainers{}

	// Setup PostgreSQL container
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("stormlink_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	containers.PostgresContainer = postgresContainer

	// Get PostgreSQL DSN
	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		containers.Cleanup()
		return nil, fmt.Errorf("failed to get postgres DSN: %w", err)
	}
	containers.PostgresDSN = dsn

	// Setup Redis container
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(wait.ForLog("Ready to accept connections")),
	)
	if err != nil {
		containers.Cleanup()
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	containers.RedisContainer = redisContainer

	// Get Redis URL
	redisURL, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		containers.Cleanup()
		return nil, fmt.Errorf("failed to get redis URL: %w", err)
	}
	containers.RedisURL = redisURL

	return containers, nil
}

// Cleanup terminates all containers
func (tc *TestContainers) Cleanup() {
	if tc.PostgresContainer != nil {
		tc.PostgresContainer.Terminate(context.Background())
	}
	if tc.RedisContainer != nil {
		tc.RedisContainer.Terminate(context.Background())
	}
}

// GetPostgresDSN returns the PostgreSQL connection string
func (tc *TestContainers) GetPostgresDSN() string {
	return tc.PostgresDSN
}

// GetRedisURL returns the Redis connection string
func (tc *TestContainers) GetRedisURL() string {
	return tc.RedisURL
}
