package httpx_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerRuntime_MatchRoute_ResolvesParameterizedPath(t *testing.T) {
	server := newServer()

	type input struct {
		ID int `path:"id"`
	}

	err := Get(server, "/users/{id}", func(_ context.Context, in *input) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	assert.NoError(t, err)

	route, ok := server.MatchRoute(http.MethodGet, "/users/42")
	assert.True(t, ok)
	assert.Equal(t, http.MethodGet, route.Method)
	assert.Equal(t, "/users/{id}", route.Path)
}
