package eventx

import (
	"log/slog"
	"reflect"
	"sync"

	"github.com/DaiYuANg/archgo/collectionx"
	"github.com/DaiYuANg/archgo/observabilityx"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/lo"
)

// Bus is an in-memory strongly typed event bus.
type Bus struct {
	lifecycleMu   sync.Mutex
	closed        bool
	nextID        uint64
	subsByType    subscriptionTable
	parallel      bool
	middleware    []Middleware
	onAsyncErr    asyncErrorHandler
	antsPool      *ants.Pool
	asyncQueue    chan publishTask
	workerWG      sync.WaitGroup
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
	lo.ForEach(opts, func(opt Option, _ int) {
		if opt != nil {
			opt(&cfg)
		}
	})

	b := &Bus{
		subsByType:    collectionx.NewConcurrentTable[reflect.Type, uint64, *subscription](),
		parallel:      cfg.parallel,
		middleware:    cfg.middleware,
		onAsyncErr:    cfg.onAsyncError,
		observability: observabilityx.Normalize(cfg.observability, nil),
	}
	b.logger = b.observability.Logger().With("component", "eventx.bus")

	// Initialize ants pool if enabled
	if cfg.useAntsPool {
		poolOpts := []ants.Option{
			ants.WithPreAlloc(true),
			ants.WithNonblocking(false),
		}
		if cfg.antsMaxBlockingCalls > 0 {
			poolOpts = append(poolOpts, ants.WithMaxBlockingTasks(cfg.antsMaxBlockingCalls))
		}

		pool, err := ants.NewPool(cfg.antsPoolSize, poolOpts...)
		if err != nil {
			b.logger.Error("failed to create ants pool", "error", err)
		} else {
			b.antsPool = pool
		}
	} else if cfg.asyncWorkers > 0 && cfg.asyncQueueSize > 0 {
		// Legacy mode: use channel-based worker pool
		b.asyncQueue = make(chan publishTask, cfg.asyncQueueSize)
		for i := 0; i < cfg.asyncWorkers; i++ {
			b.workerWG.Add(1)
			go b.workerLoop()
		}
	}

	return b
}

// Close stops accepting new events, drains async queue, and waits in-flight handlers.
func (b *Bus) Close() error {
	if b == nil {
		return nil
	}

	var queue chan publishTask
	var pool *ants.Pool
	b.lifecycleMu.Lock()
	if b.closed {
		b.lifecycleMu.Unlock()
		return nil
	}
	b.closed = true
	queue = b.asyncQueue
	pool = b.antsPool
	if queue != nil {
		close(queue)
	}
	b.lifecycleMu.Unlock()

	// Release ants pool if enabled
	if pool != nil {
		pool.Release()
	}

	// Wait for legacy worker pool if enabled
	if queue != nil {
		b.workerWG.Wait()
	}
	b.dispatchWG.Wait()
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
	handler := lo.Ternary(buildHandler != nil, buildHandler(id), nil)
	b.subsByType.Put(eventType, id, &subscription{
		id:      id,
		handler: handler,
	})
	return id, nil
}

func (b *Bus) enqueueLegacyAsyncTask(task publishTask) error {
	if b == nil {
		return ErrNilBus
	}

	b.lifecycleMu.Lock()
	defer b.lifecycleMu.Unlock()

	if b.closed {
		return ErrBusClosed
	}
	if b.asyncQueue == nil {
		return errAsyncQueueUnavailable
	}

	b.dispatchWG.Add(1)
	select {
	case b.asyncQueue <- task:
		return nil
	default:
		b.dispatchWG.Done()
		return ErrAsyncQueueFull
	}
}
