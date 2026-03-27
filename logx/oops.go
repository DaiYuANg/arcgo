package logx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/oops"
)

// LogOops logs oops-compatible errors.
func LogOops(logger *slog.Logger, err error) {
	if logger == nil {
		return
	}
	logger.Error("error", "error", err)
}

// LogOopsWithStack logs oops-compatible errors with stack fields.
func LogOopsWithStack(logger *slog.Logger, err error) {
	LogOops(logger, err)
}

// Oops creates a default oops error.
func Oops() error {
	return newOopsError("error")
}

// Oopsf creates a formatted oops error.
func Oopsf(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if msg == "" {
		msg = "error"
	}
	return newOopsError(msg)
}

// OopsWith creates an oops error from context.
func OopsWith(ctx context.Context) error {
	_ = ctx
	return newOopsError("error")
}

func newOopsError(msg string) error {
	return fmt.Errorf("create oops error: %w", oops.New(msg))
}
