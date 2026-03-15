package httpx

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adapterstd "github.com/DaiYuANg/archgo/httpx/adapter/std"
	"github.com/stretchr/testify/assert"
)

func TestServer_WithPanicRecover_Enabled(t *testing.T) {
	server := newServer(WithPanicRecover(true))

	err := Get(server, "/panic", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		panic("boom")
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "panic in handler: boom")
}

func TestServer_WithPanicRecover_Disabled(t *testing.T) {
	server := newServer(WithPanicRecover(false))

	err := Get(server, "/panic", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		panic("boom")
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	assert.Panics(t, func() {
		server.ServeHTTP(rec, req)
	})
}

func TestServer_WithAccessLog_LogsRequests(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	server := newServer(
		WithLogger(logger),
		WithAccessLog(true),
	)

	type in struct {
		ID int `path:"id"`
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	output := logs.String()
	assert.True(t, strings.Contains(output, "\"msg\":\"httpx request\""))
	assert.True(t, strings.Contains(output, "\"method\":\"GET\""))
	assert.True(t, strings.Contains(output, "\"path\":\"/users/42\""))
	assert.True(t, strings.Contains(output, "\"status\":200"))
	assert.True(t, strings.Contains(output, "\"route\":\"/users/{id}\""))
}

func TestServer_WithPrintRoutes_LogsOnRegistration(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	server := newServer(
		WithLogger(logger),
		WithPrintRoutes(true),
	)

	err := Get(server, "/routes", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	output := logs.String()
	assert.Contains(t, output, "Registered routes")
	assert.Contains(t, output, "GET /routes")
}

func TestServer_WithLogger_PropagatesToAdapter(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	stdAdapter := adapterstd.New()
	server := newServer(
		WithLogger(logger),
		WithAdapter(stdAdapter),
	)

	stdAdapter.Handle(http.MethodGet, "/native", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return errors.New("native boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/native", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logs.String(), "native boom")
}
