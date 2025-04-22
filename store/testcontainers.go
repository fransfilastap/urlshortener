package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestPostgresContainer represents a PostgreSQL container for testing
type TestPostgresContainer struct {
	Container *postgres.PostgresContainer
	URI       string
}

// TestRedisContainer represents a Redis container for testing
type TestRedisContainer struct {
	Container *tcredis.RedisContainer
	URI       string
	Client    *redis.Client
}

// SetupPostgresContainer creates and starts a PostgreSQL container for testing
func SetupPostgresContainer(ctx context.Context) (*TestPostgresContainer, error) {
	// Define PostgreSQL container with wait strategy
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("urlshortener_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithStartupTimeout(time.Second*30),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	// Get connection string directly from the container
	connStr, err := pgContainer.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres connection string: %w", err)
	}

	// Add sslmode=disable to the connection string
	if connStr[len(connStr)-1] == '?' {
		connStr += "sslmode=disable"
	} else {
		connStr += "&sslmode=disable"
	}

	// Print connection string for debugging
	fmt.Printf("PostgreSQL connection string: %s\n", connStr)

	return &TestPostgresContainer{
		Container: pgContainer,
		URI:       connStr,
	}, nil
}

// SetupRedisContainer creates and starts a Redis container for testing
func SetupRedisContainer(ctx context.Context) (*TestRedisContainer, error) {
	// Define Redis container
	redisContainer, err := tcredis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	// Get connection URI
	endpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get redis endpoint: %w", err)
	}

	uri := endpoint

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: uri,
	})

	// Test connection
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &TestRedisContainer{
		Container: redisContainer,
		URI:       uri,
		Client:    client,
	}, nil
}

// Teardown stops and removes the PostgreSQL container
func (c *TestPostgresContainer) Teardown(ctx context.Context) error {
	if c.Container != nil {
		return c.Container.Terminate(ctx)
	}
	return nil
}

// Teardown stops and removes the Redis container
func (c *TestRedisContainer) Teardown(ctx context.Context) error {
	if c.Client != nil {
		c.Client.Close()
	}
	if c.Container != nil {
		return c.Container.Terminate(ctx)
	}
	return nil
}
