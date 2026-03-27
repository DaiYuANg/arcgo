package std_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	stdadapter "github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pingOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func TestAdapter_RouterServesTypedRoute(t *testing.T) {
	a := stdadapter.New(nil)
	huma.Register(a.HumaAPI(), huma.Operation{
		OperationID: "ping",
		Method:      http.MethodGet,
		Path:        "/ping",
	}, func(_ context.Context, _ *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "pong"
		return out, nil
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ping", http.NoBody)
	rec := httptest.NewRecorder()
	a.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "pong")
}
