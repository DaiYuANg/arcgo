package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adapterpkg "github.com/DaiYuANg/toolkit4go/httpx/adapter"
	"github.com/stretchr/testify/assert"
)

type PanicEndpoint struct {
	BaseEndpoint
}

func (e *PanicEndpoint) GetPanic(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	panic("boom")
}

type VariadicEndpoint struct {
	BaseEndpoint
}

func (e *VariadicEndpoint) GetVariadic(ctx context.Context, args ...string) error {
	return nil
}

type DuplicateParamEndpoint struct {
	BaseEndpoint
}

func (e *DuplicateParamEndpoint) GetDuplicate(ctx context.Context, r1 *http.Request, r2 *http.Request) error {
	return nil
}

type MultiReturnEndpoint struct {
	BaseEndpoint
}

func (e *MultiReturnEndpoint) GetMultiReturn(ctx context.Context, w http.ResponseWriter, r *http.Request) (error, error) {
	return nil, nil
}

type TagPathEndpoint struct {
	GetUsers func() `http:"POST users"`
}

type FakeFiberAdapterNoApp struct{}

func (f *FakeFiberAdapterNoApp) Name() string { return "fiber" }

func (f *FakeFiberAdapterNoApp) Handle(method, path string, handler adapterpkg.HandlerFunc) {}

func (f *FakeFiberAdapterNoApp) Group(prefix string) adapterpkg.Adapter { return f }

func (f *FakeFiberAdapterNoApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func TestServer_PanicInHandlerReturns500(t *testing.T) {
	server := NewServer()
	err := server.Register(&PanicEndpoint{})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "panic in handler getpanic")
}

func TestServer_Register_VariadicHandler(t *testing.T) {
	server := NewServer()
	err := server.Register(&VariadicEndpoint{})
	assert.Error(t, err)
	assert.True(t, IsInvalidHandlerSignature(err))
}

func TestServer_Register_DuplicateParamType(t *testing.T) {
	server := NewServer()
	err := server.Register(&DuplicateParamEndpoint{})
	assert.Error(t, err)
	assert.True(t, IsInvalidHandlerSignature(err))
}

func TestServer_Register_MultiReturn(t *testing.T) {
	server := NewServer()
	err := server.Register(&MultiReturnEndpoint{})
	assert.Error(t, err)
	assert.True(t, IsInvalidHandlerSignature(err))
}

func TestRouterGenerator_TagPathWithoutLeadingSlash(t *testing.T) {
	gen := NewRouterGenerator(GeneratorOptions{
		BasePath:   "/api",
		UseComment: false,
		UseTag:     true,
		UseNaming:  false,
		TagKey:     "route",
	})

	routes, err := gen.Generate(&TagPathEndpoint{}).Get()
	assert.NoError(t, err)
	assert.Len(t, routes, 1)
	assert.Equal(t, http.MethodPost, routes[0].Method)
	assert.Equal(t, "/api/users", routes[0].Path)
}

func TestServer_RegisterWithPrefix_NormalizedPath(t *testing.T) {
	server := NewServer(WithBasePath("/api/"))
	err := server.RegisterWithPrefix("/v1/", &TestServerEndpoint{})
	assert.NoError(t, err)

	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/items"))
	assert.True(t, server.HasRoute(http.MethodPost, "/api/v1/item"))
}

func TestServer_ListenAndServe_FiberWithoutApp(t *testing.T) {
	server := NewServer(WithAdapter(&FakeFiberAdapterNoApp{}))
	err := server.ListenAndServe(":0")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAdapterNotFound))
}
