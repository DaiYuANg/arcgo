package redis

import (
	"errors"
	"fmt"

	"github.com/DaiYuANg/arcgo/kvx"
	goredis "github.com/redis/go-redis/v9"
)

func wrapRedisError(op string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("redis %s: %w", op, err)
}

func wrapRedisNilResult[T any](op string, value T, err error) (T, error) {
	if err == nil {
		return value, nil
	}

	var zero T
	if errors.Is(err, goredis.Nil) {
		return zero, kvx.ErrNil
	}

	return zero, wrapRedisError(op, err)
}

func wrapRedisResult[T any](op string, value T, err error) (T, error) {
	if err == nil {
		return value, nil
	}

	var zero T
	return zero, wrapRedisError(op, err)
}
