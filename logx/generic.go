package logx

import (
	"log/slog"

	"github.com/samber/lo"
)

// WithFieldT adds one typed field to logger and returns derived logger.
func WithFieldT[T any](logger *slog.Logger, key string, value T) *slog.Logger {
	if logger == nil {
		return nil
	}
	return WithField(logger, key, value)
}

// WithFieldsT adds typed fields to logger and returns derived logger.
func WithFieldsT[T any](logger *slog.Logger, fields map[string]T) *slog.Logger {
	if logger == nil {
		return nil
	}
	converted := lo.MapValues(fields, func(value T, _ string) any { return value })
	return WithFields(logger, converted)
}
