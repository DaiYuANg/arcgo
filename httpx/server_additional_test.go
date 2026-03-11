package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

type fakeFiberAdapterNoApp struct{}

func (f *fakeFiberAdapterNoApp) Name() string { return "fiber" }

func (f *fakeFiberAdapterNoApp) Handle(method, path string, handler adapter.HandlerFunc) {}

func (f *fakeFiberAdapterNoApp) Group(prefix string) adapter.Adapter { return f }

func (f *fakeFiberAdapterNoApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func (f *fakeFiberAdapterNoApp) HumaAPI() huma.API { return nil }

func (f *fakeFiberAdapterNoApp) Listen(addr string) error { return ErrAdapterNotFound }

type fakeAdapterWithoutHuma struct{}

func (f *fakeAdapterWithoutHuma) Name() string { return "fake" }

func (f *fakeAdapterWithoutHuma) Handle(method, path string, handler adapter.HandlerFunc) {}

func (f *fakeAdapterWithoutHuma) Group(prefix string) adapter.Adapter { return f }

func (f *fakeAdapterWithoutHuma) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func (f *fakeAdapterWithoutHuma) HumaAPI() huma.API { return nil }

func TestServer_GenericHandlerReturnsHTTPXError(t *testing.T) {
	server := newServer()
	err := Get(server, "/forbidden", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		return nil, NewError(http.StatusForbidden, "no permission")
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/forbidden", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "no permission")
}

func TestServer_GenericHandlerPanic(t *testing.T) {
	server := newServer()
	err := Get(server, "/panic", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		panic("boom")
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "panic in handler")
}

func TestServer_GenericHandlerNilOutputReturnsNoContent(t *testing.T) {
	server := newServer()
	err := Get(server, "/empty", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		return nil, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_AdapterWithoutHumaAPI(t *testing.T) {
	server := newServer(
		WithAdapter(&fakeAdapterWithoutHuma{}),
	)

	err := Get(server, "/huma", func(ctx context.Context, input *struct{}) (*humaPingOutput, error) {
		out := &humaPingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.Error(t, err)
	assert.ErrorContains(t, err, ErrAdapterNotFound.Error())
}

func TestServer_ListenAndServe_FiberWithoutApp(t *testing.T) {
	server := newServer(WithAdapter(&fakeFiberAdapterNoApp{}))
	err := server.ListenAndServe(":0")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAdapterNotFound))
}
