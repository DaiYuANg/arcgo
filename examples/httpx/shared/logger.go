// Package shared provides reusable helpers for the httpx examples.
package shared

import (
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/logx"
)

// NewLogger builds a common example logger and returns a cleanup function.
func NewLogger() (*slog.Logger, func(), error) {
	base, err := logx.New(logx.WithConsole(true), logx.WithDebugLevel())
	if err != nil {
		return nil, nil, fmt.Errorf("new logger: %w", err)
	}

	return base, func() {
		if closeErr := logx.Close(base); closeErr != nil {
			panic(closeErr)
		}
	}, nil
}
