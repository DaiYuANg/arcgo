package httpx_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

type endpointSpecInput struct{}

type endpointSpecOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type usersGroupEndpoint struct {
	BaseEndpoint
}

func (e *usersGroupEndpoint) EndpointSpec() EndpointSpec {
	return EndpointSpec{
		Prefix:        "/api/v1/users",
		Tags:          []string{"users", "v1"},
		Security:      []map[string][]string{{"apiKey": {}}},
		SummaryPrefix: "Users",
		Description:   "User endpoint operations",
		Parameters: []*huma.Param{{
			Name:   "X-Tenant-Id",
			In:     "header",
			Schema: &huma.Schema{Type: "string"},
		}},
		ExternalDocs: &huma.ExternalDocs{
			URL: "https://example.com/users",
		},
		Extensions: map[string]any{
			"x-endpoint": "users",
		},
	}
}

func (e *usersGroupEndpoint) RegisterGroupRoutes(group *Group) {
	MustGroupGet(group, "", func(_ context.Context, _ *endpointSpecInput) (*endpointSpecOutput, error) {
		out := &endpointSpecOutput{}
		out.Body.Message = "ok"
		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "List"
	})
}

func TestServer_RegisterOnly_GroupEndpointSpecAppliesScopedDefaults(t *testing.T) {
	server := newServer()
	server.RegisterSecurityScheme("apiKey", &huma.SecurityScheme{
		Type: "apiKey",
		Name: "X-API-Key",
		In:   "header",
	})

	server.RegisterOnly(&usersGroupEndpoint{})

	req := newTestRequest(http.MethodGet, "/api/v1/users", nil)
	rec := serveRequest(t, server, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"message":"ok"`)
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/users"))

	pathItem := server.OpenAPI().Paths["/api/v1/users"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Contains(t, pathItem.Get.Tags, "users")
		assert.Contains(t, pathItem.Get.Tags, "v1")
		assert.Equal(t, []map[string][]string{{"apiKey": {}}}, pathItem.Get.Security)
		assert.Equal(t, "Users List", pathItem.Get.Summary)
		assert.Equal(t, "User endpoint operations", pathItem.Get.Description)
		if assert.NotNil(t, pathItem.Get.ExternalDocs) {
			assert.Equal(t, "https://example.com/users", pathItem.Get.ExternalDocs.URL)
		}
		if assert.Len(t, pathItem.Get.Parameters, 1) {
			assert.Equal(t, "X-Tenant-Id", pathItem.Get.Parameters[0].Name)
			assert.Equal(t, "header", pathItem.Get.Parameters[0].In)
		}
		assert.Equal(t, "users", pathItem.Get.Extensions["x-endpoint"])
	}
}
