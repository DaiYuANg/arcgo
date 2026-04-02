package httpx_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ssePingData struct {
	Message string `json:"message"`
}

func TestServer_GetSSE_StreamsMessages(t *testing.T) {
	server := newServer()

	err := GetSSE(server, "/events", map[string]any{
		"ping": ssePingData{},
	}, func(_ context.Context, _ *struct{}, send SSESender) {
		require.NoError(t, send.Data(ssePingData{Message: "pong"}))
	})
	assert.NoError(t, err)

	req := newTestRequest(http.MethodGet, "/events", nil)
	rec := serveRequest(t, server, req)

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
	}, func(_ context.Context, _ *struct{}, send SSESender) {
		require.NoError(t, send.Data(ssePingData{Message: "hello"}))
	})
	assert.NoError(t, err)

	req := newTestRequest(http.MethodGet, "/api/v1/events", nil)
	rec := serveRequest(t, server, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")
	assert.Contains(t, rec.Body.String(), `"message":"hello"`)
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/events"))
	assert.Equal(t, 1, server.GetRoutesByPath("/api/v1").Len())

	pathItem := server.OpenAPI().Paths["/api/v1/events"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Contains(t, pathItem.Get.Responses["200"].Content, "text/event-stream")
	}
}

func TestServer_GetSSE_EmptyEventMap(t *testing.T) {
	server := newServer()
	err := GetSSE(server, "/events", nil, func(_ context.Context, _ *struct{}, _ SSESender) {})
	assert.Error(t, err)
	assert.ErrorContains(t, err, "sse event map is empty")
}

func TestServer_GetSSE_NilEventType(t *testing.T) {
	server := newServer()
	err := GetSSE(server, "/events", map[string]any{
		"ping": nil,
	}, func(_ context.Context, _ *struct{}, _ SSESender) {})
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
	}, func(_ context.Context, _ *struct{}, _ SSESender) {})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAdapterNotFound)
}

func TestServer_RouteSSEWithPolicies_WrapAndOperation(t *testing.T) {
	server := newServer()

	err := RouteSSEWithPolicies(server, MethodGet, "/events/policy", map[string]any{
		"ping": ssePingData{},
	}, func(_ context.Context, _ *struct{}, send SSESender) {
		require.NoError(t, send.Data(ssePingData{Message: "from-handler"}))
	}, SSERoutePolicy[struct{}]{
		Name: "prefix-message",
		Wrap: func(next SSEHandler[struct{}]) SSEHandler[struct{}] {
			return func(ctx context.Context, input *struct{}, send SSESender) {
				require.NoError(t, send.Data(ssePingData{Message: "from-policy"}))
				next(ctx, input, send)
			}
		},
	}, SSEPolicyOperation[struct{}](huma.OperationTags("sse-policy")))
	assert.NoError(t, err)

	req := newTestRequest(http.MethodGet, "/events/policy", nil)
	rec := serveRequest(t, server, req)

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
	}, func(_ context.Context, _ *struct{}, send SSESender) {
		require.NoError(t, send.Data(ssePingData{Message: "ok"}))
	})
	assert.NoError(t, err)

	req := newTestRequest(http.MethodGet, "/api/v2/events/policy", nil)
	rec := serveRequest(t, server, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v2/events/policy"))
}

func TestServer_DuplicateSSERouteRegistrationReturnsError(t *testing.T) {
	server := newServer()

	require.NoError(t, GetSSE(server, "/events", map[string]any{
		"ping": ssePingData{},
	}, func(_ context.Context, _ *struct{}, send SSESender) {
		require.NoError(t, send.Data(ssePingData{Message: "first"}))
	}))

	err := GetSSE(server, "/events", map[string]any{
		"ping": ssePingData{},
	}, func(_ context.Context, _ *struct{}, send SSESender) {
		require.NoError(t, send.Data(ssePingData{Message: "second"}))
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrRouteAlreadyExists)
	assert.Equal(t, 1, server.RouteCount())
}
