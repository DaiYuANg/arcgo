package httpx

import (
	"context"
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
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrServerFrozen)
	assert.False(t, server.HasRoute(MethodGet, "/frozen-modifier"))
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

type fakeSlowShutdownAdapter struct {
	listenStarted  chan struct{}
	shutdownCalled chan struct{}
	allowReturn    chan struct{}
}

func (f *fakeSlowShutdownAdapter) Name() string { return "slow-shutdown" }

func (f *fakeSlowShutdownAdapter) HumaAPI() huma.API { return nil }

func (f *fakeSlowShutdownAdapter) Listen(addr string) error {
	close(f.listenStarted)
	<-f.allowReturn
	return nil
}

func (f *fakeSlowShutdownAdapter) Shutdown() error {
	close(f.shutdownCalled)
	return nil
}

func TestServer_ListenAndServeContext_FallbackReturnsOnContextCancellation(t *testing.T) {
	adapter := &fakeSlowShutdownAdapter{
		listenStarted:  make(chan struct{}),
		shutdownCalled: make(chan struct{}),
		allowReturn:    make(chan struct{}),
	}
	server := newServer(WithAdapter(adapter))

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServeContext(ctx, ":0")
	}()

	select {
	case <-adapter.listenStarted:
	case <-time.After(time.Second):
		t.Fatal("listen adapter did not start in time")
	}

	cancel()

	select {
	case <-adapter.shutdownCalled:
	case <-time.After(time.Second):
		t.Fatal("shutdown was not called in time")
	}

	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("ListenAndServeContext did not return after context cancellation")
	}

	close(adapter.allowReturn)
}
