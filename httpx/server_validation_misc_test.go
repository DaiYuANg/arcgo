package httpx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestServer_StrongTypedQueryAndHeaderBinding(t *testing.T) {
	server := newServer()

	err := Get(server, "/params", func(_ context.Context, input *paramsInput) (*paramsOutput, error) {
		out := &paramsOutput{}
		out.Body.ID = input.ID
		out.Body.Flag = input.Flag
		out.Body.Trace = input.Trace
		return out, nil
	})
	assert.NoError(t, err)

	req := newTestRequest(http.MethodGet, "/params?id=42&flag=true", nil)
	req.Header.Set("X-Trace-ID", "trace-001")
	w := serveRequest(t, server, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":42`)
	assert.Contains(t, w.Body.String(), `"flag":true`)
	assert.Contains(t, w.Body.String(), `"trace":"trace-001"`)
}

func TestServer_WithMiddleware(t *testing.T) {
	// Note: Middleware must be added to the adapter before passing to httpx.Server.
	// Huma is now initialized at adapter creation time, so middleware should be
	// configured on the router/engine before calling adapter.New().
	server := newServer()
	err := Get(server, "/items", func(_ context.Context, _ *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	req := newTestRequest(http.MethodGet, "/items", nil)
	w := serveRequest(t, server, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestServer_DefaultHumaEnabled(t *testing.T) {
	server := newServer()

	err := Get(server, "/huma", func(_ context.Context, _ *struct{}) (*humaPingOutput, error) {
		out := &humaPingOutput{}
		out.Body.Message = "from huma"
		return out, nil
	}, huma.OperationTags("demo"))
	assert.NoError(t, err)

	req := newTestRequest(http.MethodGet, "/huma", nil)
	w := serveRequest(t, server, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "from huma")
	assert.NotNil(t, server.HumaAPI())
}

func TestServer_WithValidation_WorksWithHuma(t *testing.T) {
	server := newServer(
		WithValidation(),
	)

	err := Get(server, "/validate-huma", func(_ context.Context, _ *validatedQueryInput) (*humaPingOutput, error) {
		out := &humaPingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	req := newTestRequest(http.MethodGet, "/validate-huma", nil)
	w := serveRequest(t, server, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "request validation failed")
}

func TestServer_WithCustomValidator(t *testing.T) {
	customValidator := validator.New()
	err := customValidator.RegisterValidation("arc", func(fl validator.FieldLevel) bool {
		return fl.Field().String() == "arc"
	})
	assert.NoError(t, err)

	server := newServer(WithValidator(customValidator))

	err = Post(server, "/custom-validate", func(_ context.Context, input *customValidatedInput) (*validatedBodyOutput, error) {
		out := &validatedBodyOutput{}
		out.Body.Name = input.Body.Name
		return out, nil
	})
	assert.NoError(t, err)

	req := newTestRequest(http.MethodPost, "/custom-validate", bytes.NewReader([]byte(`{"name":"bad"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := serveRequest(t, server, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "request validation failed")
}

func TestServer_GetRoutesAndFilters(t *testing.T) {
	server := newServer()

	err := Get(server, "/users", func(_ context.Context, _ *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	routes := server.GetRoutes()
	assert.Equal(t, 1, routes.Len())
	firstRoute, ok := routes.Get(0)
	assert.True(t, ok)
	assert.Equal(t, http.MethodGet, firstRoute.Method)

	getRoutes := server.GetRoutesByMethod(http.MethodGet)
	assert.Equal(t, 1, getRoutes.Len())
	mutatedRoute, ok := getRoutes.Get(0)
	assert.True(t, ok)
	mutatedRoute.Path = "/mutated"
	assert.True(t, getRoutes.Set(0, mutatedRoute))
	refreshedGetRoutes := server.GetRoutesByMethod(http.MethodGet)
	refreshedRoute, ok := refreshedGetRoutes.Get(0)
	assert.True(t, ok)
	assert.Equal(t, "/users", refreshedRoute.Path)

	grouped := server.GetRoutesGroupedByMethod()
	assert.Len(t, grouped.Get(http.MethodGet), 1)
	grouped.Set(http.MethodGet)
	assert.Equal(t, 1, server.GetRoutesByMethod(http.MethodGet).Len())

	pathRoutes := server.GetRoutesByPath("/users")
	assert.Equal(t, 1, pathRoutes.Len())

	assert.True(t, server.HasRoute(http.MethodGet, "/users"))

	var resp map[string]any
	req := newTestRequest(http.MethodGet, "/users", nil)
	w := serveRequest(t, server, req)
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
}
