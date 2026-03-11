package shared

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/logx"
)

// NewLogger builds a common example logger and returns a cleanup function.
func NewLogger() (*slog.Logger, func(), error) {
	base, err := logx.New(logx.WithConsole(true), logx.WithDebugLevel())
	if err != nil {
		return nil, nil, err
	}

	return logx.NewSlog(base), func() {
		_ = base.Close()
	}, nil
}
