package logx_test

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/DaiYuANg/arcgo/logx"
)

type simpleCloser interface {
	Close() error
}

type failingCloser struct {
	err error
}

func (c *failingCloser) Close() error {
	return c.err
}

type closableHandler struct {
	slog.Handler
	closers []simpleCloser
	once    sync.Once
	err     error
}

func (h *closableHandler) Close() error {
	h.once.Do(func() {
		errs := make([]error, 0, len(h.closers))
		for _, closer := range h.closers {
			if closer == nil {
				continue
			}
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close test logger resource: %w", err))
			}
		}
		h.err = errors.Join(errs...)
	})

	return h.err
}

func newTestLogger(tb testing.TB, opts ...logx.Option) *slog.Logger {
	tb.Helper()

	logger, err := logx.New(opts...)
	if err != nil {
		tb.Fatal(err)
	}

	tb.Cleanup(func() {
		if err := logx.Close(logger); err != nil {
			tb.Errorf("Close() error = %v", err)
		}
	})

	return logger
}

var _ io.Closer = (*failingCloser)(nil)
