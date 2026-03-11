package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

type ssePingData struct {
	Message string `json:"message"`
}

func TestServer_GetSSE_StreamsMessages(t *testing.T) {
	server := newServer()

	err := GetSSE(server, "/events", map[string]any{
		"ping": ssePingData{},
	}, func(ctx context.Context, input *struct{}, send SSESender) {
		_ = send.Data(ssePingData{Message: "pong"})
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")
	assert.Contains(t, rec.Body.String(), "event: ping")
	assert.Contains(t, rec.Body.String(), `"message":"pong"`)
	assert.True(t, server.HasRoute(http.MethodGet, "/events"))

	pathItem := server.OpenAPI().Paths["/events"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		response := pathItem.Get.Responses["200"]
		if assert.NotNil(t, response) {
			assert.Contains(t, response.Content, "text/event-stream")
		}
	}
}

func TestServer_GroupGetSSE_WithBasePath(t *testing.T) {
	server := newServer(WithBasePath("/api"))
	group := server.Group("/v1")

	err := GroupGetSSE(group, "/events", map[string]any{
		"message": ssePingData{},
	}, func(ctx context.Context, input *struct{}, send SSESender) {
		_ = send.Data(ssePingData{Message: "hello"})
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")
	assert.Contains(t, rec.Body.String(), `"message":"hello"`)
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/events"))
	assert.Len(t, server.GetRoutesByPath("/api/v1"), 1)

	pathItem := server.OpenAPI().Paths["/api/v1/events"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Contains(t, pathItem.Get.Responses["200"].Content, "text/event-stream")
	}
}

func TestServer_GetSSE_EmptyEventMap(t *testing.T) {
	server := newServer()
	err := GetSSE(server, "/events", nil, func(ctx context.Context, input *struct{}, send SSESender) {})
	assert.Error(t, err)
	assert.ErrorContains(t, err, "sse event map is empty")
}

func TestServer_GetSSE_NilEventType(t *testing.T) {
	server := newServer()
	err := GetSSE(server, "/events", map[string]any{
		"ping": nil,
	}, func(ctx context.Context, input *struct{}, send SSESender) {})
	assert.Error(t, err)
	assert.ErrorContains(t, err, "sse event type is nil")
}

func TestServer_GetSSE_NilHandler(t *testing.T) {
	server := newServer()
	var handler SSEHandler[struct{}]

	err := GetSSE(server, "/events", map[string]any{
		"ping": ssePingData{},
	}, handler)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "sse handler is nil")
}

func TestServer_GetSSE_AdapterWithoutHumaAPI(t *testing.T) {
	server := newServer(WithAdapter(&fakeAdapterWithoutHuma{}))

	err := GetSSE(server, "/events", map[string]any{
		"ping": ssePingData{},
	}, func(ctx context.Context, input *struct{}, send SSESender) {})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAdapterNotFound)
}

func TestServer_RouteSSEWithPolicies_WrapAndOperation(t *testing.T) {
	server := newServer()

	err := RouteSSEWithPolicies(server, MethodGet, "/events/policy", map[string]any{
		"ping": ssePingData{},
	}, func(ctx context.Context, input *struct{}, send SSESender) {
		_ = send.Data(ssePingData{Message: "from-handler"})
	}, SSERoutePolicy[struct{}]{
		Name: "prefix-message",
		Wrap: func(next SSEHandler[struct{}]) SSEHandler[struct{}] {
			return func(ctx context.Context, input *struct{}, send SSESender) {
				_ = send.Data(ssePingData{Message: "from-policy"})
				next(ctx, input, send)
			}
		},
	}, SSEPolicyOperation[struct{}](huma.OperationTags("sse-policy")))
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/events/policy", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"message":"from-policy"`)
	assert.Contains(t, rec.Body.String(), `"message":"from-handler"`)
	pathItem := server.OpenAPI().Paths["/events/policy"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Contains(t, pathItem.Get.Tags, "sse-policy")
	}
}

func TestServer_GroupRouteSSEWithPolicies_WithBasePath(t *testing.T) {
	server := newServer(WithBasePath("/api"))
	group := server.Group("/v2")

	err := GroupRouteSSEWithPolicies(group, MethodGet, "/events/policy", map[string]any{
		"ping": ssePingData{},
	}, func(ctx context.Context, input *struct{}, send SSESender) {
		_ = send.Data(ssePingData{Message: "ok"})
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/events/policy", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v2/events/policy"))
}
