package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	adapterstd "github.com/DaiYuANg/archgo/httpx/adapter/std"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

func TestServer_ConfigureDocs_RebindsRoutesAtRuntime(t *testing.T) {
	server := newServer()

	server.ConfigureDocs(func(d *DocsOptions) {
		d.DocsPath = "/reference"
		d.OpenAPIPath = "/spec"
		d.SchemasPath = "/contracts"
		d.Renderer = DocsRendererSwaggerUI
	})

	oldDocsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	oldDocsRec := httptest.NewRecorder()
	server.ServeHTTP(oldDocsRec, oldDocsReq)
	assert.Equal(t, http.StatusNotFound, oldDocsRec.Code)

	newDocsReq := httptest.NewRequest(http.MethodGet, "/reference", nil)
	newDocsRec := httptest.NewRecorder()
	server.ServeHTTP(newDocsRec, newDocsReq)
	assert.Equal(t, http.StatusOK, newDocsRec.Code)
	assert.Contains(t, newDocsRec.Body.String(), "swagger-ui")

	oldSpecReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	oldSpecRec := httptest.NewRecorder()
	server.ServeHTTP(oldSpecRec, oldSpecReq)
	assert.Equal(t, http.StatusNotFound, oldSpecRec.Code)

	newSpecReq := httptest.NewRequest(http.MethodGet, "/spec.json", nil)
	newSpecRec := httptest.NewRecorder()
	server.ServeHTTP(newSpecRec, newSpecReq)
	assert.Equal(t, http.StatusOK, newSpecRec.Code)
}

func TestServer_ConfigureDocs_WithExternalAdapter(t *testing.T) {
	stdAdapter := adapterstd.New()
	server := newServer(WithAdapter(stdAdapter))

	server.ConfigureDocs(func(d *DocsOptions) {
		d.Enabled = false
	})

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGroup_DefaultParametersSummaryAndDescription(t *testing.T) {
	server := newServer()
	group := server.Group("/reports")
	group.DefaultParameters(&huma.Param{
		Name:        "X-Tenant",
		In:          "header",
		Description: "tenant header",
		Schema:      &huma.Schema{Type: "string"},
	})
	group.DefaultSummaryPrefix("Reports")
	group.DefaultDescription("Shared reporting endpoints")

	err := GroupGet(group, "/daily", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
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
	group.RegisterTags(
		&huma.Tag{Name: "admin", Description: "Administrative endpoints"},
		&huma.Tag{Name: "ops", Description: "Operations"},
	)
	group.DefaultTags("admin", "ops")
	group.DefaultExternalDocs(&huma.ExternalDocs{
		Description: "Admin guide",
		URL:         "https://example.com/admin",
	})
	group.DefaultExtensions(map[string]any{
		"x-group": "admin",
	})

	err := GroupGet(group, "/health", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
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
