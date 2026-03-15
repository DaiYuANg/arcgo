package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	adapterecho "github.com/DaiYuANg/archgo/httpx/adapter/echo"
	adapterfiber "github.com/DaiYuANg/archgo/httpx/adapter/fiber"
	adaptergin "github.com/DaiYuANg/archgo/httpx/adapter/gin"
	"github.com/stretchr/testify/assert"
)

func TestServer_StrongTypedPathBindingOnStdAdapter(t *testing.T) {
	server := newServer()

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		Body struct {
			ID int `json:"id"`
		}
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*out, error) {
		result := &out{}
		result.Body.ID = input.UserID
		return result, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":123`)
}

func TestServer_StrongTypedPathBindingOnGinAdapter(t *testing.T) {
	server := newServer(WithAdapter(adaptergin.New(nil)))

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		Body struct {
			ID int `json:"id"`
		}
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*out, error) {
		result := &out{}
		result.Body.ID = input.UserID
		return result, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/88", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":88`)
}

func TestServer_StrongTypedPathBindingOnEchoAdapter(t *testing.T) {
	server := newServer(WithAdapter(adapterecho.New(nil)))

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		Body struct {
			ID int `json:"id"`
		}
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*out, error) {
		result := &out{}
		result.Body.ID = input.UserID
		return result, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/77", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":77`)
}

func TestServer_StrongTypedPathBindingOnFiberAdapter(t *testing.T) {
	server := newServer(WithAdapter(adapterfiber.New(nil)))

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		Body struct {
			ID int `json:"id"`
		}
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*out, error) {
		result := &out{}
		result.Body.ID = input.UserID
		return result, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/66", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
}
