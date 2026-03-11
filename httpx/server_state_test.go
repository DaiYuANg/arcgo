package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ListenAndServeContext_FreezesConfiguration(t *testing.T) {
	ctxAdapter := &fakeContextAdapter{
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
	server := newServer(WithAdapter(ctxAdapter))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServeContext(ctx, ":0")
	}()

	select {
	case <-ctxAdapter.started:
	case <-time.After(time.Second):
		t.Fatal("listen context adapter did not start in time")
	}
	assert.True(t, server.IsFrozen())

	cancel()
	select {
	case <-ctxAdapter.stopped:
	case <-time.After(time.Second):
		t.Fatal("listen context adapter did not stop in time")
	}

	require.NoError(t, <-errCh)
}

func TestServer_FrozenConfig_DoesNotAcceptOperationModifier(t *testing.T) {
	server := newServer()
	server.freezeConfiguration()

	server.UseOperationModifier(func(op *huma.Operation) {
		op.Tags = append(op.Tags, "blocked")
	})

	err := Get(server, "/frozen-modifier", func(ctx context.Context, input *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.Message = "ok"
		return out, nil
	})
	require.NoError(t, err)

	path := server.OpenAPI().Paths["/frozen-modifier"]
	require.NotNil(t, path)
	require.NotNil(t, path.Get)
	assert.NotContains(t, path.Get.Tags, "blocked")
}

func TestServer_FrozenConfig_DoesNotReconfigureDocs(t *testing.T) {
	server := newServer()
	server.freezeConfiguration()

	server.ConfigureDocs(func(d *DocsOptions) {
		d.DocsPath = "/reference"
		d.OpenAPIPath = "/spec"
	})

	docs := server.Docs()
	assert.Equal(t, "/docs", docs.DocsPath)
	assert.Equal(t, "/openapi", docs.OpenAPIPath)

	req := httptest.NewRequest(http.MethodGet, "/reference", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServer_UseOpenAPIPatch_RespectsFreeze(t *testing.T) {
	server := newServer()

	server.UseOpenAPIPatch(func(doc *huma.OpenAPI) {
		doc.Info.Title = "patched-before-freeze"
	})
	assert.Equal(t, "patched-before-freeze", server.OpenAPI().Info.Title)

	server.freezeConfiguration()
	server.UseOpenAPIPatch(func(doc *huma.OpenAPI) {
		doc.Info.Title = "patched-after-freeze"
	})

	assert.Equal(t, "patched-before-freeze", server.OpenAPI().Info.Title)
}
