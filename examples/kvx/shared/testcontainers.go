package shared

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const defaultServerPort = "6379/tcp"

func RedisImage() string {
	return envOrDefault("KVX_REDIS_IMAGE", "redis:7-alpine")
}

func RedisJSONImage() string {
	return envOrDefault("KVX_REDIS_JSON_IMAGE", "redis/redis-stack-server:latest")
}

func ValkeyImage() string {
	return envOrDefault("KVX_VALKEY_IMAGE", "valkey/valkey:8-alpine")
}

func ValkeyJSONImage() string {
	return envOrDefault("KVX_VALKEY_JSON_IMAGE", "valkey/valkey:8-alpine")
}

func StartContainer(ctx context.Context, image string) (testcontainers.Container, string, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{defaultServerPort},
			WaitingFor:   wait.ForListeningPort(defaultServerPort).WithStartupTimeout(45 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return nil, "", err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, "", err
	}

	port, err := container.MappedPort(ctx, defaultServerPort)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, "", err
	}

	return container, fmt.Sprintf("%s:%s", host, port.Port()), nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
