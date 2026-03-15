package httpx

import (
	"context"
	"net/http"
	"testing"

	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

type testFeatureAdapter interface {
	EnableFeature(name string)
}

type fakeFeatureAdapter struct {
	feature string
}

func (f *fakeFeatureAdapter) Name() string { return "feature" }

func (f *fakeFeatureAdapter) Handle(method, path string, handler adapter.HandlerFunc) {}

func (f *fakeFeatureAdapter) Group(prefix string) adapter.Adapter { return f }

func (f *fakeFeatureAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func (f *fakeFeatureAdapter) HumaAPI() huma.API { return nil }

func (f *fakeFeatureAdapter) EnableFeature(name string) {
	f.feature = name
}

func TestUseAdapter_CustomCapability(t *testing.T) {
	a := &fakeFeatureAdapter{}
	server := newServer(WithAdapter(a))

	called := UseAdapter[testFeatureAdapter](server, func(feature testFeatureAdapter) {
		feature.EnableFeature("streaming")
	})

	assert.True(t, called)
	assert.Equal(t, "streaming", a.feature)
}

func TestUseAdapter_NotSupported(t *testing.T) {
	server := newServer()
	called := UseAdapter[testFeatureAdapter](server, func(feature testFeatureAdapter) {
		feature.EnableFeature("streaming")
	})
	assert.False(t, called)
}

type fakeContextAdapter struct {
	started chan struct{}
	stopped chan struct{}
}

func (f *fakeContextAdapter) Name() string { return "ctx-adapter" }

func (f *fakeContextAdapter) Handle(method, path string, handler adapter.HandlerFunc) {}

func (f *fakeContextAdapter) Group(prefix string) adapter.Adapter { return f }

func (f *fakeContextAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func (f *fakeContextAdapter) HumaAPI() huma.API { return nil }

func (f *fakeContextAdapter) ListenContext(ctx context.Context, addr string) error {
	close(f.started)
	<-ctx.Done()
	close(f.stopped)
	return nil
}
