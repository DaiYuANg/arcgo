package httpx_test

import (
	"context"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

func TestGroup_DefaultParametersSummaryAndDescription(t *testing.T) {
	server := newServer()
	group := server.Group("/reports")
	group.DefaultParameters(Parameters(&huma.Param{
		Name:        "X-Tenant",
		In:          "header",
		Description: "tenant header",
		Schema:      &huma.Schema{Type: "string"},
	}))
	group.DefaultSummaryPrefix("Reports")
	group.DefaultDescription("Shared reporting endpoints")

	err := GroupGet(group, "/daily", func(_ context.Context, _ *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "Daily usage"
	})
	assert.NoError(t, err)

	pathItem := server.OpenAPI().Paths["/reports/daily"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Equal(t, "Reports Daily usage", pathItem.Get.Summary)
		assert.Equal(t, "Shared reporting endpoints", pathItem.Get.Description)
		if assert.Len(t, pathItem.Get.Parameters, 1) {
			assert.Equal(t, "X-Tenant", pathItem.Get.Parameters[0].Name)
			assert.Equal(t, "header", pathItem.Get.Parameters[0].In)
		}
	}
}

func TestGroup_RegisterTagsExternalDocsAndExtensions(t *testing.T) {
	server := newServer()
	group := server.Group("/admin")
	group.RegisterTags(TagDefinitions(
		&huma.Tag{Name: "admin", Description: "Administrative endpoints"},
		&huma.Tag{Name: "ops", Description: "Operations"},
	))
	group.DefaultTags(Tags("admin", "ops"))
	group.DefaultExternalDocs(&huma.ExternalDocs{
		Description: "Admin guide",
		URL:         "https://example.com/admin",
	})
	group.DefaultExtensions(Extensions(map[string]any{
		"x-group": "admin",
	}))

	err := GroupGet(group, "/health", func(_ context.Context, _ *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	doc := server.OpenAPI()
	if assert.NotNil(t, doc) {
		assert.GreaterOrEqual(t, len(doc.Tags), 2)
	}

	pathItem := doc.Paths["/admin/health"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Contains(t, pathItem.Get.Tags, "admin")
		assert.Contains(t, pathItem.Get.Tags, "ops")
		if assert.NotNil(t, pathItem.Get.ExternalDocs) {
			assert.Equal(t, "https://example.com/admin", pathItem.Get.ExternalDocs.URL)
		}
		assert.Equal(t, "admin", pathItem.Get.Extensions["x-group"])
	}
}
