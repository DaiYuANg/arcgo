package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	adapterecho "github.com/DaiYuANg/arcgo/httpx/adapter/echo"
	adapterfiber "github.com/DaiYuANg/arcgo/httpx/adapter/fiber"
	adaptergin "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

type pingOutput struct {
	Message string `json:"message"`
}

type echoInput struct {
	Name string `json:"name"`
}

type echoOutput struct {
	Name string `json:"name"`
}

type customBindInput struct {
	ID    int
	Token string
}

func (i *customBindInput) BindRequest(r *http.Request) error {
	if r == nil {
		return nil
	}
	rawID := r.URL.Query().Get("user_id")
	if rawID == "" {
		return nil
	}

	id, err := strconv.Atoi(rawID)
	if err != nil {
		return fmt.Errorf("parse user_id: %w", err)
	}
	i.ID = id
	i.Token = r.Header.Get("X-Token")
	return nil
}

type customBindOutput struct {
	ID    int    `json:"id"`
	Token string `json:"token"`
}

type paramsInput struct {
	ID      int           `query:"id"`
	Flag    bool          `query:"flag"`
	Timeout time.Duration `query:"timeout"`
	IDs     []int         `query:"ids"`
	Trace   string        `header:"X-Trace-ID"`
}

type paramsOutput struct {
	ID      int    `json:"id"`
	Flag    bool   `json:"flag"`
	Timeout string `json:"timeout"`
	IDs     []int  `json:"ids"`
	Trace   string `json:"trace"`
}

type humaPingOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func TestServer_GenericGetWithoutHuma(t *testing.T) {
	server := NewServer()

	err := Get(server, "/ping", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		return &pingOutput{Message: "pong"}, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pong")
	assert.False(t, server.HasHuma())
}

func TestServer_GenericPostDecodeBody(t *testing.T) {
	server := NewServer()

	err := Post(server, "/echo", func(ctx context.Context, input *echoInput) (*echoOutput, error) {
		return &echoOutput{Name: input.Name}, nil
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
	server := NewServer()

	err := Post(server, "/echo", func(ctx context.Context, input *echoInput) (*echoOutput, error) {
		return &echoOutput{Name: input.Name}, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader([]byte(`{"name":`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request input")
}

func TestServer_CustomRequestBinder(t *testing.T) {
	server := NewServer()

	err := Get(server, "/custom-bind", func(ctx context.Context, input *customBindInput) (*customBindOutput, error) {
		return &customBindOutput{
			ID:    input.ID,
			Token: input.Token,
		}, nil
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
	server := NewServer()

	err := Get(server, "/custom-bind", func(ctx context.Context, input *customBindInput) (*customBindOutput, error) {
		return &customBindOutput{
			ID: input.ID,
		}, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/custom-bind?user_id=not-an-int", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request input")
}

func TestServer_GroupWithBasePath(t *testing.T) {
	server := NewServer(WithBasePath("/api"))
	v1 := server.Group("/v1")

	err := GroupGet(v1, "/health", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		return &pingOutput{Message: "ok"}, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, server.HasRoute(http.MethodGet, "/api/v1/health"))
}

func TestServer_StrongTypedQueryAndHeaderBinding(t *testing.T) {
	server := NewServer()

	err := Get(server, "/params", func(ctx context.Context, input *paramsInput) (*paramsOutput, error) {
		return &paramsOutput{
			ID:      input.ID,
			Flag:    input.Flag,
			Timeout: input.Timeout.String(),
			IDs:     input.IDs,
			Trace:   input.Trace,
		}, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/params?id=42&flag=true&timeout=3s&ids=1,2,3", nil)
	req.Header.Set("X-Trace-ID", "trace-001")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":42`)
	assert.Contains(t, w.Body.String(), `"flag":true`)
	assert.Contains(t, w.Body.String(), `"timeout":"3s"`)
	assert.Contains(t, w.Body.String(), `"ids":[1,2,3]`)
	assert.Contains(t, w.Body.String(), `"trace":"trace-001"`)
}

func TestServer_StrongTypedPathBindingOnStdAdapter(t *testing.T) {
	server := NewServer()

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		ID int `json:"id"`
	}

	err := Get(server, "/users/{id}", func(ctx context.Context, input *in) (*out, error) {
		return &out{ID: input.UserID}, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":123`)
}

func TestServer_StrongTypedPathBindingOnGinAdapter(t *testing.T) {
	server := NewServer(WithAdapter(adaptergin.New()))

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		ID int `json:"id"`
	}

	err := Get(server, "/users/:id", func(ctx context.Context, input *in) (*out, error) {
		return &out{ID: input.UserID}, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/88", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":88`)
}

func TestServer_StrongTypedPathBindingOnEchoAdapter(t *testing.T) {
	server := NewServer(WithAdapter(adapterecho.New()))

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		ID int `json:"id"`
	}

	err := Get(server, "/users/:id", func(ctx context.Context, input *in) (*out, error) {
		return &out{ID: input.UserID}, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/77", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":77`)
}

func TestServer_StrongTypedPathBindingOnFiberAdapter(t *testing.T) {
	server := NewServer(WithAdapter(adapterfiber.New()))

	type in struct {
		UserID int `path:"id"`
	}
	type out struct {
		ID int `json:"id"`
	}

	err := Get(server, "/users/:id", func(ctx context.Context, input *in) (*out, error) {
		return &out{ID: input.UserID}, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/users/66", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestServer_WithMiddleware(t *testing.T) {
	var middlewareCalled bool

	stdAdapter := std.New()
	stdAdapter.Router().Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	})

	server := NewServer(WithAdapter(stdAdapter))
	err := Get(server, "/items", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		return &pingOutput{Message: "ok"}, nil
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.True(t, middlewareCalled)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_WithHumaEnabled(t *testing.T) {
	server := NewServer(WithHuma(HumaOptions{
		Enabled:     true,
		Title:       "ArcGo API",
		Version:     "1.0.0",
		Description: "typed api",
	}))

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
	assert.True(t, server.HasHuma())
	assert.NotNil(t, server.HumaAPI())
}

type fakeHumaAdapter struct {
	lastOpts adapter.HumaOptions
	enabled  bool
}

func (f *fakeHumaAdapter) Name() string { return "fake" }

func (f *fakeHumaAdapter) Handle(method, path string, handler adapter.HandlerFunc) {}

func (f *fakeHumaAdapter) Group(prefix string) adapter.Adapter { return f }

func (f *fakeHumaAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func (f *fakeHumaAdapter) EnableHuma(opts adapter.HumaOptions) {
	f.lastOpts = opts
	f.enabled = true
}

func (f *fakeHumaAdapter) HumaAPI() huma.API { return nil }

func (f *fakeHumaAdapter) HasHuma() bool { return f.enabled }

func TestServer_WithHumaOptionCallsAdapter(t *testing.T) {
	fake := &fakeHumaAdapter{}

	_ = NewServer(
		WithAdapter(fake),
		WithHuma(HumaOptions{
			Enabled: true,
			Title:   "Test API",
			Version: "1.0.0",
		}),
	)

	assert.True(t, fake.enabled)
	assert.Equal(t, "Test API", fake.lastOpts.Title)
	assert.Equal(t, "1.0.0", fake.lastOpts.Version)
}

func TestServer_GetRoutesAndFilters(t *testing.T) {
	server := NewServer()

	err := Get(server, "/users", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		return &pingOutput{Message: "ok"}, nil
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
