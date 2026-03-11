//go:build !no_fiber

package fiber

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	fiberframework "github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewWithOptions_AppOptionsMerge(t *testing.T) {
	a := NewWithOptions(nil, Options{
		App: AppOptions{
			ReadTimeout:     2 * time.Second,
			ShutdownTimeout: 9 * time.Second,
		},
	})

	cfg := a.Router().Config()
	assert.Equal(t, 2*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 15*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 60*time.Second, cfg.IdleTimeout)
	assert.Equal(t, 9*time.Second, a.opts.ShutdownTimeout)
}

func TestNewWithOptions_ExternalAppKeepsOwnTimeouts(t *testing.T) {
	external := fiberframework.New(fiberframework.Config{
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 2 * time.Second,
		IdleTimeout:  3 * time.Second,
	})

	a := NewWithOptions(external, Options{
		App: AppOptions{
			ReadTimeout:  9 * time.Second,
			WriteTimeout: 9 * time.Second,
			IdleTimeout:  9 * time.Second,
		},
	})

	cfg := a.Router().Config()
	assert.Equal(t, 1*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 2*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 3*time.Second, cfg.IdleTimeout)
}

func TestNewWithOptions_LoggerUsedByNativeHandler(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	a := NewWithOptions(nil, Options{
		Logger: logger,
	})
	a.Handle(http.MethodGet, "/err", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return errors.New("native boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	resp, err := a.Router().Test(req, -1)
	assert.NoError(t, err)
	if resp != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, logs.String(), "native boom")
}
