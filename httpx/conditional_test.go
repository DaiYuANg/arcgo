package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type conditionalReadInput struct {
	ConditionalParams
}

type conditionalWriteInput struct {
	ConditionalParams
}

func TestServer_ConditionalRead_NotModified304(t *testing.T) {
	server := newServer()
	modified := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)

	err := Get(server, "/resources/read", func(ctx context.Context, input *conditionalReadInput) (*pingOutput, error) {
		if err := input.PreconditionFailed("v1", modified); err != nil {
			return nil, err
		}
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	}, OperationConditionalRead())
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/resources/read", nil)
	req.Header.Set("If-None-Match", `"v1"`)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotModified, rec.Code)

	pathItem := server.OpenAPI().Paths["/resources/read"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Contains(t, pathItem.Get.Responses, "304")
	}
}

func TestServer_ConditionalWrite_PreconditionFailed412(t *testing.T) {
	server := newServer()
	modified := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)

	err := Put(server, "/resources/write", func(ctx context.Context, input *conditionalWriteInput) (*pingOutput, error) {
		if err := input.PreconditionFailed("v2", modified); err != nil {
			return nil, err
		}
		out := &pingOutput{}
		out.Body.Message = "updated"
		return out, nil
	}, OperationConditionalWrite())
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/resources/write", nil)
	req.Header.Set("If-Match", `"old-version"`)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusPreconditionFailed, rec.Code)
	assert.Contains(t, rec.Body.String(), "precondition failed")

	pathItem := server.OpenAPI().Paths["/resources/write"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Put) {
		assert.Contains(t, pathItem.Put.Responses, "412")
	}
}
