package echo

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWithOptions_ServerOptionsMerge(t *testing.T) {
	a := NewWithOptions(nil, Options{
		Server: ServerOptions{
			IdleTimeout:     45 * time.Second,
			ShutdownTimeout: 9 * time.Second,
		},
	})

	assert.Equal(t, 15*time.Second, a.server.ReadTimeout)
	assert.Equal(t, 15*time.Second, a.server.WriteTimeout)
	assert.Equal(t, 45*time.Second, a.server.IdleTimeout)
	assert.Equal(t, 9*time.Second, a.server.ShutdownTimeout)
	assert.Equal(t, 1<<20, a.server.MaxHeaderBytes)
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
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logs.String(), "native boom")
}
