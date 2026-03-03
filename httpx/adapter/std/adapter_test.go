package std

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdapter_HandleBasic(t *testing.T) {
	a := New()
	a.Handle(http.MethodGet, "/ping", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusNoContent)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAdapter_GroupPrefix(t *testing.T) {
	a := New()
	group := a.Group("/api")
	group.Handle(http.MethodGet, "/health", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestAdapter_HandleError(t *testing.T) {
	a := New()
	a.Handle(http.MethodGet, "/err", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return errors.New("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "boom")
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		path   string
		want   string
	}{
		{name: "empty both", prefix: "", path: "", want: "/"},
		{name: "empty prefix", prefix: "", path: "ping", want: "/ping"},
		{name: "normal slash", prefix: "/api", path: "/ping", want: "/api/ping"},
		{name: "no slash path", prefix: "/api", path: "ping", want: "/api/ping"},
		{name: "root path", prefix: "/api", path: "/", want: "/api"},
		{name: "trim right slash", prefix: "/api/", path: "/ping", want: "/api/ping"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, joinPath(tt.prefix, tt.path))
		})
	}
}
