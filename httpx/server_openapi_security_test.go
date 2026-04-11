package httpx_test

import (
	"context"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

func TestServer_SecurityComponentsAndGlobalHeader(t *testing.T) {
	server := newServer(
		WithSecurity(SecurityOptions{
			Schemes: SecuritySchemes(map[string]*huma.SecurityScheme{
				"bearerAuth": {
					Type:   "http",
					Scheme: "bearer",
				},
			}),
			Requirements: SecurityRequirements(
				SecurityRequirement("bearerAuth"),
			),
		}),
		WithGlobalHeaders(Parameters(&huma.Param{
			Name:        "X-Request-Id",
			In:          "header",
			Description: "request correlation id",
			Schema:      &huma.Schema{Type: "string"},
		})),
	)

	server.RegisterComponentParameter("Locale", &huma.Param{
		Name:   "locale",
		In:     "query",
		Schema: &huma.Schema{Type: "string"},
	})
	server.RegisterComponentHeader("RateLimit", &huma.Header{
		Description: "rate limit",
		Schema:      &huma.Schema{Type: "integer"},
	})

	err := Get(server, "/secure", func(_ context.Context, _ *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	doc := server.OpenAPI()
	if assert.NotNil(t, doc) && assert.NotNil(t, doc.Components) {
		assert.Contains(t, doc.Components.SecuritySchemes, "bearerAuth")
		assert.Contains(t, doc.Components.Parameters, "Locale")
		assert.Contains(t, doc.Components.Headers, "RateLimit")
		assert.Equal(t, []map[string][]string{{"bearerAuth": {}}}, doc.Security)
	}

	pathItem := doc.Paths["/secure"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		if assert.Len(t, pathItem.Get.Parameters, 1) {
			assert.Equal(t, "X-Request-Id", pathItem.Get.Parameters[0].Name)
			assert.Equal(t, "header", pathItem.Get.Parameters[0].In)
			if assert.NotNil(t, pathItem.Get.Parameters[0].Schema) {
				assert.Equal(t, "string", pathItem.Get.Parameters[0].Schema.Type)
			}
		}
	}
}

func TestServer_RegisterGlobalParameter_ClonesSchema(t *testing.T) {
	server := newServer()
	param := &huma.Param{
		Name:   "X-Clone",
		In:     "header",
		Schema: &huma.Schema{Type: "string"},
	}
	server.RegisterGlobalParameter(param)

	// Mutating the original parameter after registration should not affect future operations.
	param.Schema.Type = "integer"

	err := Get(server, "/clone-param", func(_ context.Context, _ *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	pathItem := server.OpenAPI().Paths["/clone-param"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		if assert.Len(t, pathItem.Get.Parameters, 1) {
			if assert.NotNil(t, pathItem.Get.Parameters[0].Schema) {
				assert.Equal(t, "string", pathItem.Get.Parameters[0].Schema.Type)
			}
		}
	}
}

func TestGroup_DefaultTagsAndSecurity(t *testing.T) {
	server := newServer()
	server.RegisterSecurityScheme("apiKey", &huma.SecurityScheme{
		Type: "apiKey",
		Name: "X-API-Key",
		In:   "header",
	})

	group := server.Group("/admin")
	group.DefaultTags(Tags("admin", "protected"))
	group.DefaultSecurity(SecurityRequirements(SecurityRequirement("apiKey")))

	err := GroupGet(group, "/stats", func(_ context.Context, _ *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	pathItem := server.OpenAPI().Paths["/admin/stats"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Contains(t, pathItem.Get.Tags, "admin")
		assert.Contains(t, pathItem.Get.Tags, "protected")
		assert.Equal(t, []map[string][]string{{"apiKey": {}}}, pathItem.Get.Security)
	}
}
