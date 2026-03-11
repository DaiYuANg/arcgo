package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

type policyConditionalInput struct {
	ConditionalParams
}

type policyBinaryOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

func TestServer_RouteWithPolicies_ConditionalRead(t *testing.T) {
	server := newServer()
	modified := time.Date(2026, 3, 11, 9, 0, 0, 0, time.UTC)

	err := RouteWithPolicies(server, MethodGet, "/policy/read", func(ctx context.Context, input *policyConditionalInput) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	}, PolicyConditionalRead[policyConditionalInput, pingOutput](func(ctx context.Context, input *policyConditionalInput) (string, time.Time, error) {
		return "etag-v1", modified, nil
	}))
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/policy/read", nil)
	req.Header.Set("If-None-Match", `"etag-v1"`)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotModified, rec.Code)
	assert.True(t, server.HasRoute(http.MethodGet, "/policy/read"))
	assert.Contains(t, server.OpenAPI().Paths["/policy/read"].Get.Responses, "304")
}

func TestServer_RouteWithPolicies_ConditionalWrite(t *testing.T) {
	server := newServer()
	modified := time.Date(2026, 3, 11, 9, 0, 0, 0, time.UTC)

	err := RouteWithPolicies(server, MethodPut, "/policy/write", func(ctx context.Context, input *policyConditionalInput) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "updated"
		return out, nil
	}, PolicyConditionalWrite[policyConditionalInput, pingOutput](func(ctx context.Context, input *policyConditionalInput) (string, time.Time, error) {
		return "etag-v2", modified, nil
	}))
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/policy/write", nil)
	req.Header.Set("If-Match", `"old"`)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusPreconditionFailed, rec.Code)
	assert.Contains(t, server.OpenAPI().Paths["/policy/write"].Put.Responses, "412")
}

func TestServer_RouteWithPolicies_HTMLResponse(t *testing.T) {
	server := newServer()

	err := RouteWithPolicies(server, MethodGet, "/policy/html", func(ctx context.Context, input *struct{}) (*policyBinaryOutput, error) {
		return &policyBinaryOutput{
			Body: []byte("<h1>hello</h1>"),
		}, nil
	}, PolicyHTMLResponse[struct{}, policyBinaryOutput]())
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/policy/html", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "<h1>hello</h1>")
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, server.OpenAPI().Paths["/policy/html"].Get.Responses["200"].Content, "text/html")
}

func TestServer_RouteWithPolicies_ImageResponse(t *testing.T) {
	server := newServer()

	err := RouteWithPolicies(server, MethodGet, "/policy/image", func(ctx context.Context, input *struct{}) (*policyBinaryOutput, error) {
		return &policyBinaryOutput{
			Body: []byte("img"),
		}, nil
	}, PolicyImageResponse[struct{}, policyBinaryOutput]("image/png", "image/jpeg"), PolicyOperation[struct{}, policyBinaryOutput](huma.OperationTags("media")))
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/policy/image", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "img", rec.Body.String())
	assert.Contains(t, rec.Header().Get("Content-Type"), "image/png")
	pathItem := server.OpenAPI().Paths["/policy/image"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Contains(t, pathItem.Get.Responses["200"].Content, "image/png")
		assert.Contains(t, pathItem.Get.Responses["200"].Content, "image/jpeg")
		assert.Contains(t, pathItem.Get.Tags, "media")
	}
}
