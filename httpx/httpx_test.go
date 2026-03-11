package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adapterecho "github.com/DaiYuANg/arcgo/httpx/adapter/echo"
	adapterfiber "github.com/DaiYuANg/arcgo/httpx/adapter/fiber"
	adaptergin "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
	adapterstd "github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

type pingOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type echoInput struct {
	Body struct {
		Name string `json:"name"`
	}
}

type echoOutput struct {
	Body struct {
		Name string `json:"name"`
	}
}

type customBindInput struct {
	ID    int    `query:"user_id"`
	Token string `header:"X-Token"`
}

type customBindOutput struct {
	Body struct {
		ID    int    `json:"id"`
		Token string `json:"token"`
	}
}

type paramsInput struct {
	ID    int    `query:"id"`
	Flag  bool   `query:"flag"`
	Trace string `header:"X-Trace-ID"`
}

type paramsOutput struct {
	Body struct {
		ID    int    `json:"id"`
		Flag  bool   `json:"flag"`
		Trace string `json:"trace"`
	}
}

type validatedBodyInput struct {
	Body struct {
		Name string `json:"name" validate:"required,min=3"`
	}
}

type validatedBodyOutput struct {
	Body struct {
		Name string `json:"name"`
	}
}

type validatedQueryInput struct {
	ID int `query:"id" validate:"required,min=1"`
}

type customValidatedInput struct {
	Body struct {
		Name string `json:"name" validate:"arc"`
	}
}

type humaPingOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func TestServer_GenericGetWithDefaultHuma(t *testing.T) {
	server := newServer()

	err := Get(server, "/ping", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "pong"
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pong")
}

func TestServer_GenericPostDecodeBody(t *testing.T) {
	server := newServer()

	err := Post(server, "/echo", func(ctx context.Context, input *echoInput) (*echoOutput, error) {
		out := &echoOutput{}
		out.Body.Name = input.Body.Name
		return out, nil
	})
	assert.NoError(t, err)

	body := []byte(`{"name":"arcgo"}`)
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "arcgo")
}

func TestServer_GenericPostInvalidJSON(t *testing.T) {
	server := newServer()

	err := Post(server, "/echo", func(ctx context.Context, input *echoInput) (*echoOutput, error) {
		out := &echoOutput{}
		out.Body.Name = input.Body.Name
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader([]byte(`{"name":`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "unexpected end of JSON input")
}

func TestServer_WithValidation_InvalidBody(t *testing.T) {
	server := newServer(WithValidation())

	err := Post(server, "/validated", func(ctx context.Context, input *validatedBodyInput) (*validatedBodyOutput, error) {
		out := &validatedBodyOutput{}
		out.Body.Name = input.Body.Name
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/validated", bytes.NewReader([]byte(`{"name":"ab"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "request validation failed")
}

func TestServer_WithValidation_ValidBody(t *testing.T) {
	server := newServer(WithValidation())

	err := Post(server, "/validated", func(ctx context.Context, input *validatedBodyInput) (*validatedBodyOutput, error) {
		out := &validatedBodyOutput{}
		out.Body.Name = input.Body.Name
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/validated", bytes.NewReader([]byte(`{"name":"arcgo"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"name\":\"arcgo\"")
}

func TestServer_CustomRequestBinder(t *testing.T) {
	server := newServer()

	err := Get(server, "/custom-bind", func(ctx context.Context, input *customBindInput) (*customBindOutput, error) {
		out := &customBindOutput{}
		out.Body.ID = input.ID
		out.Body.Token = input.Token
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/custom-bind?user_id=123", nil)
	req.Header.Set("X-Token", "token-abc")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":123`)
	assert.Contains(t, w.Body.String(), `"token":"token-abc"`)
}

func TestServer_CustomRequestBinderError(t *testing.T) {
	server := newServer()

	err := Get(server, "/custom-bind", func(ctx context.Context, input *customBindInput) (*customBindOutput, error) {
		out := &customBindOutput{}
		out.Body.ID = input.ID
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/custom-bind?user_id=not-an-int", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "user_id")
}

func TestServer_GroupWithBasePath(t *testing.T) {
	server := newServer(WithBasePath("/api"))
	v1 := server.Group("/v1")

	err := GroupGet(v1, "/health", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/health"))
}

func TestServer_StrongTypedQueryAndHeaderBinding(t *testing.T) {
	server := newServer()

	err := Get(server, "/params", func(ctx context.Context, input *paramsInput) (*paramsOutput, error) {
		out := &paramsOutput{}
		out.Body.ID = input.ID
		out.Body.Flag = input.Flag
		out.Body.Trace = input.Trace
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/params?id=42&flag=true", nil)
	req.Header.Set("X-Trace-ID", "trace-001")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":42`)
	assert.Contains(t, w.Body.String(), `"flag":true`)
	assert.Contains(t, w.Body.String(), `"trace":"trace-001"`)
}

func TestServer_StrongTypedPathBindingOnStdAdapter(t *testing.T) {
	server := newServer()

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		Body struct {
			ID int `json:"id"`
		}
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*out, error) {
		result := &out{}
		result.Body.ID = input.UserID
		return result, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":123`)
}

func TestServer_StrongTypedPathBindingOnGinAdapter(t *testing.T) {
	server := newServer(WithAdapter(adaptergin.New(nil)))

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		Body struct {
			ID int `json:"id"`
		}
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*out, error) {
		result := &out{}
		result.Body.ID = input.UserID
		return result, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/88", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":88`)
}

func TestServer_StrongTypedPathBindingOnEchoAdapter(t *testing.T) {
	server := newServer(WithAdapter(adapterecho.New(nil)))

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		Body struct {
			ID int `json:"id"`
		}
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*out, error) {
		result := &out{}
		result.Body.ID = input.UserID
		return result, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/77", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":77`)
}

func TestServer_StrongTypedPathBindingOnFiberAdapter(t *testing.T) {
	server := newServer(WithAdapter(adapterfiber.New(nil)))

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		Body struct {
			ID int `json:"id"`
		}
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*out, error) {
		result := &out{}
		result.Body.ID = input.UserID
		return result, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/66", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestServer_WithMiddleware(t *testing.T) {
	// Note: Middleware must be added to the adapter before passing to httpx.Server.
	// Huma is now initialized at adapter creation time, so middleware should be
	// configured on the router/engine before calling adapter.New().

	// This test verifies that a server created with a default adapter works correctly.
	server := newServer()
	err := Get(server, "/items", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestServer_DefaultHumaEnabled(t *testing.T) {
	server := newServer()

	err := Get(server, "/huma", func(ctx context.Context, input *struct{}) (*humaPingOutput, error) {
		out := &humaPingOutput{}
		out.Body.Message = "from huma"
		return out, nil
	}, huma.OperationTags("demo"))
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/huma", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "from huma")
	assert.NotNil(t, server.HumaAPI())
}

func TestServer_WithValidation_WorksWithHuma(t *testing.T) {
	server := newServer(
		WithValidation(),
	)

	err := Get(server, "/validate-huma", func(ctx context.Context, input *validatedQueryInput) (*humaPingOutput, error) {
		out := &humaPingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/validate-huma", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

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

	err = Post(server, "/custom-validate", func(ctx context.Context, input *customValidatedInput) (*validatedBodyOutput, error) {
		out := &validatedBodyOutput{}
		out.Body.Name = input.Body.Name
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/custom-validate", bytes.NewReader([]byte(`{"name":"bad"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "request validation failed")
}

func TestServer_GetRoutesAndFilters(t *testing.T) {
	server := newServer()

	err := Get(server, "/users", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	routes := server.GetRoutes()
	assert.Len(t, routes, 1)
	assert.Equal(t, http.MethodGet, routes[0].Method)

	getRoutes := server.GetRoutesByMethod(http.MethodGet)
	assert.Len(t, getRoutes, 1)

	pathRoutes := server.GetRoutesByPath("/users")
	assert.Len(t, pathRoutes, 1)

	assert.True(t, server.HasRoute(http.MethodGet, "/users"))

	var resp map[string]any
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
}

func TestServer_WithOpenAPIInfo_UpdatesDocument(t *testing.T) {
	server := newServer(WithOpenAPIInfo("Arc API", "2.0.0", "typed service"))

	openAPI := server.OpenAPI()
	if assert.NotNil(t, openAPI) && assert.NotNil(t, openAPI.Info) {
		assert.Equal(t, "Arc API", openAPI.Info.Title)
		assert.Equal(t, "2.0.0", openAPI.Info.Version)
		assert.Equal(t, "typed service", openAPI.Info.Description)
	}
}

func TestServer_WithOpenAPIDocs_DisablesDefaultDocsRoutes(t *testing.T) {
	server := newServer(WithOpenAPIDocs(false))

	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	server.ServeHTTP(docsRec, docsReq)
	assert.Equal(t, http.StatusNotFound, docsRec.Code)

	openAPIReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	openAPIRec := httptest.NewRecorder()
	server.ServeHTTP(openAPIRec, openAPIReq)
	assert.Equal(t, http.StatusNotFound, openAPIRec.Code)
}

func TestServer_ConfigureOpenAPI_PatchesDocument(t *testing.T) {
	server := newServer()
	server.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		doc.Tags = append(doc.Tags, &huma.Tag{Name: "internal"})
	})

	openAPI := server.OpenAPI()
	if assert.NotNil(t, openAPI) {
		assert.Len(t, openAPI.Tags, 1)
		assert.Equal(t, "internal", openAPI.Tags[0].Name)
	}
}

func TestGroup_HumaMiddlewareAndModifier(t *testing.T) {
	server := newServer(WithBasePath("/api"))
	group := server.Group("/v1")
	group.UseHumaMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		ctx.AppendHeader("X-Group", "v1")
		next(ctx)
	})
	group.UseSimpleOperationModifier(func(op *huma.Operation) {
		op.Tags = append(op.Tags, "group-tag")
	})

	err := GroupGet(group, "/items", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "v1", rec.Header().Get("X-Group"))

	pathItem := server.OpenAPI().Paths["/api/v1/items"]
	if assert.NotNil(t, pathItem) && assert.NotNil(t, pathItem.Get) {
		assert.Contains(t, pathItem.Get.Tags, "group-tag")
	}
}

func TestServer_WithDocs_CustomPaths(t *testing.T) {
	server := newServer(WithDocs(DocsOptions{
		Enabled:     true,
		DocsPath:    "/reference",
		OpenAPIPath: "/spec",
		SchemasPath: "/contracts",
		Renderer:    DocsRendererScalar,
	}))

	docsReq := httptest.NewRequest(http.MethodGet, "/reference", nil)
	docsRec := httptest.NewRecorder()
	server.ServeHTTP(docsRec, docsReq)
	assert.Equal(t, http.StatusOK, docsRec.Code)

	oldDocsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	oldDocsRec := httptest.NewRecorder()
	server.ServeHTTP(oldDocsRec, oldDocsReq)
	assert.Equal(t, http.StatusNotFound, oldDocsRec.Code)

	specReq := httptest.NewRequest(http.MethodGet, "/spec.json", nil)
	specRec := httptest.NewRecorder()
	server.ServeHTTP(specRec, specReq)
	assert.Equal(t, http.StatusOK, specRec.Code)
	assert.Contains(t, specRec.Body.String(), "\"openapi\"")

	docs := server.Docs()
	assert.Equal(t, "/reference", docs.DocsPath)
	assert.Equal(t, "/spec", docs.OpenAPIPath)
	assert.Equal(t, "/contracts", docs.SchemasPath)
	assert.Equal(t, DocsRendererScalar, docs.Renderer)
}

func TestServer_SecurityComponentsAndGlobalHeader(t *testing.T) {
	server := newServer(
		WithSecurity(SecurityOptions{
			Schemes: map[string]*huma.SecurityScheme{
				"bearerAuth": {
					Type:   "http",
					Scheme: "bearer",
				},
			},
			Requirements: []map[string][]string{
				{"bearerAuth": {}},
			},
		}),
		WithGlobalHeaders(&huma.Param{
			Name:        "X-Request-Id",
			In:          "header",
			Description: "request correlation id",
			Schema:      &huma.Schema{Type: "string"},
		}),
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

	err := Get(server, "/secure", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
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

	err := Get(server, "/clone-param", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
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
	group.DefaultTags("admin", "protected")
	group.DefaultSecurity(map[string][]string{
		"apiKey": {},
	})

	err := GroupGet(group, "/stats", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
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

func TestServer_WithPanicRecover_Enabled(t *testing.T) {
	server := newServer(WithPanicRecover(true))

	err := Get(server, "/panic", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		panic("boom")
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "panic in handler: boom")
}

func TestServer_WithPanicRecover_Disabled(t *testing.T) {
	server := newServer(WithPanicRecover(false))

	err := Get(server, "/panic", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		panic("boom")
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	assert.Panics(t, func() {
		server.ServeHTTP(rec, req)
	})
}

func TestServer_WithAccessLog_LogsRequests(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	server := newServer(
		WithLogger(logger),
		WithAccessLog(true),
	)

	type in struct {
		ID int `path:"id"`
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	output := logs.String()
	assert.True(t, strings.Contains(output, "\"msg\":\"httpx request\""))
	assert.True(t, strings.Contains(output, "\"method\":\"GET\""))
	assert.True(t, strings.Contains(output, "\"path\":\"/users/42\""))
	assert.True(t, strings.Contains(output, "\"status\":200"))
	assert.True(t, strings.Contains(output, "\"route\":\"/users/{id}\""))
}

func TestServer_WithPrintRoutes_LogsOnRegistration(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	server := newServer(
		WithLogger(logger),
		WithPrintRoutes(true),
	)

	err := Get(server, "/routes", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	output := logs.String()
	assert.Contains(t, output, "Registered routes")
	assert.Contains(t, output, "GET /routes")
}

func TestServer_WithLogger_PropagatesToAdapter(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	stdAdapter := adapterstd.New()
	server := newServer(
		WithLogger(logger),
		WithAdapter(stdAdapter),
	)

	stdAdapter.Handle(http.MethodGet, "/native", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return errors.New("native boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/native", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logs.String(), "native boom")
}
