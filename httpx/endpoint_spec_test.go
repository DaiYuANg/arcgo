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

type usersRegistrarEndpoint struct{}
type legacyRoutesEndpoint struct {
	BaseEndpoint
}

func (e *usersGroupEndpoint) EndpointSpec() EndpointSpec {
	return EndpointSpec{
		Prefix:        "/api/v1/users",
		Tags:          Tags("users", "v1"),
		Security:      SecurityRequirements(SecurityRequirement("apiKey")),
		SummaryPrefix: "Users",
		Description:   "User endpoint operations",
		Parameters: Parameters(&huma.Param{
			Name:   "X-Tenant-Id",
			In:     "header",
			Schema: &huma.Schema{Type: "string"},
		}),
		ExternalDocs: &huma.ExternalDocs{
			URL: "https://example.com/users",
		},
		Extensions: Extensions(map[string]any{
			"x-endpoint": "users",
		}),
	}
}

func (e *usersRegistrarEndpoint) EndpointSpec() EndpointSpec {
	return EndpointSpec{
		Prefix:        "/api/v2/users",
		Tags:          Tags("users", "v2"),
		SummaryPrefix: "Users",
		Description:   "Registrar endpoint operations",
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

func (e *usersRegistrarEndpoint) Register(registrar Registrar) {
	group := registrar.Scope()

	MustGroupGet(group, "", func(_ context.Context, _ *endpointSpecInput) (*endpointSpecOutput, error) {
		out := &endpointSpecOutput{}
		out.Body.Message = "ok"
		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "List"
	})

	audit := group.Group("/audit")
	MustGroupGet(audit, "", func(_ context.Context, _ *endpointSpecInput) (*endpointSpecOutput, error) {
		out := &endpointSpecOutput{}
		out.Body.Message = "audit"
		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "Audit"
	})
}

func (e *legacyRoutesEndpoint) RegisterRoutes(server ServerRuntime) {
	MustGet(server, "/legacy/users", func(_ context.Context, _ *endpointSpecInput) (*endpointSpecOutput, error) {
		out := &endpointSpecOutput{}
		out.Body.Message = "legacy"
		return out, nil
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

func TestServer_RegisterOnly_RegistrarEndpointUsesScopedGroup(t *testing.T) {
	server := newServer()

	server.RegisterOnly(&usersRegistrarEndpoint{})

	req := newTestRequest(http.MethodGet, "/api/v2/users/audit", nil)
	rec := serveRequest(t, server, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"message":"audit"`)
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v2/users"))
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v2/users/audit"))

	rootPathItem := server.OpenAPI().Paths["/api/v2/users"]
	if assert.NotNil(t, rootPathItem) && assert.NotNil(t, rootPathItem.Get) {
		assert.Contains(t, rootPathItem.Get.Tags, "users")
		assert.Contains(t, rootPathItem.Get.Tags, "v2")
		assert.Equal(t, "Users List", rootPathItem.Get.Summary)
		assert.Equal(t, "Registrar endpoint operations", rootPathItem.Get.Description)
	}

	auditPathItem := server.OpenAPI().Paths["/api/v2/users/audit"]
	if assert.NotNil(t, auditPathItem) && assert.NotNil(t, auditPathItem.Get) {
		assert.Contains(t, auditPathItem.Get.Tags, "users")
		assert.Contains(t, auditPathItem.Get.Tags, "v2")
		assert.Equal(t, "Users Audit", auditPathItem.Get.Summary)
		assert.Equal(t, "Registrar endpoint operations", auditPathItem.Get.Description)
	}
}

func TestServer_RegisterOnly_LegacyEndpointStillWorks(t *testing.T) {
	server := newServer()

	server.RegisterOnly(&legacyRoutesEndpoint{})

	req := newTestRequest(http.MethodGet, "/legacy/users", nil)
	rec := serveRequest(t, server, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"message":"legacy"`)
	assert.True(t, server.HasRoute(http.MethodGet, "/legacy/users"))
}
