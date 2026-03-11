package httpx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

func TestServer_ConcurrentModifiersAndRouteRegistration(t *testing.T) {
	server := newServer()

	const total = 120
	var wg sync.WaitGroup
	errCh := make(chan error, total)

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			server.UseOperationModifier(func(op *huma.Operation) {
				op.Tags = append(op.Tags, fmt.Sprintf("mod-%d", index))
			})
		}(i)
	}

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			path := fmt.Sprintf("/concurrent/%d", index)
			err := Get(server, path, func(ctx context.Context, input *struct{}) (*pingOutput, error) {
				out := &pingOutput{}
				out.Body.Message = "ok"
				return out, nil
			})
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		assert.NoError(t, err)
	}

	assert.Equal(t, total, server.RouteCount())

	req := httptest.NewRequest(http.MethodGet, "/concurrent/0", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
