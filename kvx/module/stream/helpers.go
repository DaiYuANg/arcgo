package stream

import (
	"context"
	"encoding/json"
	"fmt"
)

func wrapError(err error, action string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return nil
}

func wrapResult[T any](value T, err error, action string) (T, error) {
	if err != nil {
		var zero T
		return zero, fmt.Errorf("%s: %w", action, err)
	}

	return value, nil
}

func wrapContextError(ctx context.Context, action string) error {
	return fmt.Errorf("%s: %w", action, ctx.Err())
}

func marshalJSON(v any, action string) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}

	return data, nil
}
