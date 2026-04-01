package eventx

import (
	"log/slog"
	"reflect"
	"sync"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/DaiYuANg/arcgo/pkg/option"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/mo"
)

// Bus is an in-memory strongly typed event bus.
type Bus struct {
	lifecycleMu   sync.Mutex
	closed        bool
	nextID        uint64
	subsByType    subscriptionTable
	handlerCache  collectionx.ConcurrentMap[reflect.Type, []HandlerFunc]
	parallel      bool
	parallelLimit chan struct{}
	middleware    []Middleware
	onAsyncErr    asyncErrorHandler
	antsPool      *ants.Pool
	initErr       error
	dispatchWG    sync.WaitGroup
	observability observabilityx.Observability
	logger        *slog.Logger
}

const (
	metricDispatchTotal           = "eventx_dispatch_total"
	metricDispatchDurationMS      = "eventx_dispatch_duration_ms"
	metricAsyncEnqueueTotal       = "eventx_async_enqueue_total"
	metricAsyncEnqueueDurationMS  = "eventx_async_enqueue_duration_ms"
	metricAsyncDispatchErrorTotal = "eventx_async_dispatch_error_total"
)

// New creates a new Bus runtime.
func New(opts ...Option) BusRuntime {
	cfg := defaultOptions()
	option.Apply(&cfg, opts...)

	b := &Bus{
		subsByType:    collectionx.NewConcurrentTable[reflect.Type, uint64, *subscription](),
		handlerCache:  collectionx.NewConcurrentMap[reflect.Type, []HandlerFunc](),
		parallel:      cfg.parallel,
		parallelLimit: newParallelLimiter(cfg.antsPoolSize),
		middleware:    cfg.middleware,
		onAsyncErr:    cfg.onAsyncError,
		observability: observabilityx.Normalize(cfg.observability, nil),
	}
	b.logger = b.observability.Logger().With("component", "eventx.bus")

	poolOpts := collectionx.NewListWithCapacity[ants.Option](3,
		ants.WithPreAlloc(true),
		ants.WithNonblocking(false),
	)
	if cfg.antsMaxBlockingCalls > 0 {
		poolOpts.Add(ants.WithMaxBlockingTasks(cfg.antsMaxBlockingCalls))
	}

	pool, err := ants.NewPool(cfg.antsPoolSize, poolOpts.Values()...)
	if err != nil {
		b.initErr = err
		b.logger.Error("failed to create ants pool", "error", err)
	} else {
		b.antsPool = pool
	}

	b.logger.Debug("bus initialized",
		"parallel", b.parallel,
		"ants_pool_size", cfg.antsPoolSize,
		"ants_max_blocking_calls", cfg.antsMaxBlockingCalls,
		"middleware_count", len(cfg.middleware),
		"async_runtime_ready", b.antsPool != nil,
	)

	return b
}

// Close stops accepting new events, drains async queue, and waits in-flight handlers.
func (b *Bus) Close() error {
	if b == nil {
		return nil
	}

	var pool *ants.Pool
	b.lifecycleMu.Lock()
	if b.closed {
		b.lifecycleMu.Unlock()
		return nil
	}
	b.closed = true
	pool = b.antsPool
	b.lifecycleMu.Unlock()

	b.logger.Debug("closing bus",
		"subscriber_count", b.subsByType.Len(),
	)

	if pool != nil {
		pool.Release()
	}
	b.dispatchWG.Wait()
	b.logger.Debug("bus closed")
	return nil
}

// SubscriberCount returns active subscriber count.
func (b *Bus) SubscriberCount() int {
	if b == nil {
		return 0
	}
	return b.subsByType.Len()
}

func (b *Bus) beginDispatch() bool {
	if b == nil {
		return false
	}

	b.lifecycleMu.Lock()
	defer b.lifecycleMu.Unlock()
	if b.closed {
		return false
	}
	b.dispatchWG.Add(1)
	return true
}

func (b *Bus) registerSubscription(eventType reflect.Type, buildHandler func(id uint64) HandlerFunc) (uint64, error) {
	if b == nil {
		return 0, ErrNilBus
	}

	b.lifecycleMu.Lock()
	defer b.lifecycleMu.Unlock()
	if b.closed {
		return 0, ErrBusClosed
	}
	b.nextID++
	id := b.nextID
	handler := HandlerFunc(nil)
	if factory, ok := mo.TupleToOption(buildHandler, buildHandler != nil).Get(); ok {
		handler = factory(id)
	}
	b.subsByType.Put(eventType, id, &subscription{
		id:      id,
		handler: handler,
	})
	b.invalidateHandlerSnapshot(eventType)
	b.logger.Debug("subscription registered",
		"event_type", eventType.String(),
		"subscription_id", id,
	)
	return id, nil
}

func (b *Bus) invalidateHandlerSnapshot(eventType reflect.Type) {
	if b == nil || b.handlerCache == nil {
		return
	}
	b.handlerCache.Delete(eventType)
}

func (b *Bus) deleteSubscription(eventType reflect.Type, id uint64) {
	if b == nil {
		return
	}
	if b.subsByType.Delete(eventType, id) {
		b.invalidateHandlerSnapshot(eventType)
		b.logger.Debug("subscription removed",
			"event_type", eventType.String(),
			"subscription_id", id,
		)
	}
}

func newParallelLimiter(size int) chan struct{} {
	if size <= 0 {
		return nil
	}
	return make(chan struct{}, size)
}
