package httpx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeHooks struct {
	startFn func()
	stopFn  func()
}

func (f *fakeHooks) OnStart(fn func()) { f.startFn = fn }

func (f *fakeHooks) OnStop(fn func()) { f.stopFn = fn }

func TestBindGracefulShutdownHooks(t *testing.T) {
	hooks := &fakeHooks{}
	adapter := &fakeContextAdapter{
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
	server := newServer(WithAdapter(adapter))

	BindGracefulShutdownHooks(hooks, server, ":0")
	if assert.NotNil(t, hooks.startFn) && assert.NotNil(t, hooks.stopFn) {
		go hooks.startFn()
		<-adapter.started
		hooks.stopFn()

		select {
		case <-adapter.stopped:
		case <-time.After(time.Second):
			t.Fatal("expected adapter to stop after hook stop callback")
		}
	}
}
