package eventx

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/DaiYuANg/archgo/observabilityx"
)

// Publish dispatches one event synchronously to all matching subscribers.
func (b *Bus) Publish(ctx context.Context, event Event) error {
	if b == nil {
		return ErrNilBus
	}
	if event == nil {
		return ErrNilEvent
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if !b.beginDispatch() {
		return ErrBusClosed
	}
	defer b.dispatchWG.Done()

	handlers := b.snapshotHandlersByEventType(reflect.TypeOf(event))

	return b.dispatch(ctx, event, handlers, "sync")
}

// PublishAsync enqueues one event for asynchronous dispatch.
func (b *Bus) PublishAsync(ctx context.Context, event Event) error {
	if b == nil {
		return ErrNilBus
	}
	if event == nil {
		return ErrNilEvent
	}
	if ctx == nil {
		ctx = context.Background()
	}

	obs := b.observabilitySafe()
	start := time.Now()
	ctx, span := obs.StartSpan(ctx, "eventx.publish.async.enqueue",
		observabilityx.String("event_name", eventName(event)),
	)
	defer span.End()

	handlers := b.snapshotHandlersByEventType(reflect.TypeOf(event))

	// Use ants pool if enabled
	if b.antsPool != nil {
		if !b.beginDispatch() {
			err := ErrBusClosed
			span.RecordError(err)
			obs.AddCounter(ctx, metricAsyncEnqueueTotal, 1,
				observabilityx.String("result", "closed"),
				observabilityx.String("event_name", eventName(event)),
			)
			obs.RecordHistogram(ctx, metricAsyncEnqueueDurationMS, float64(time.Since(start).Milliseconds()),
				observabilityx.String("result", "closed"),
				observabilityx.String("event_name", eventName(event)),
			)
			return err
		}

		task := publishTask{
			ctx:      ctx,
			event:    event,
			handlers: handlers,
		}
		err := b.antsPool.Submit(func() {
			defer b.dispatchWG.Done()
			b.executeTask(task)
		})
		if err != nil {
			b.dispatchWG.Done()
			span.RecordError(err)
			obs.AddCounter(ctx, metricAsyncEnqueueTotal, 1,
				observabilityx.String("result", "pool_error"),
				observabilityx.String("event_name", eventName(event)),
			)
			obs.RecordHistogram(ctx, metricAsyncEnqueueDurationMS, float64(time.Since(start).Milliseconds()),
				observabilityx.String("result", "pool_error"),
				observabilityx.String("event_name", eventName(event)),
			)
			return fmt.Errorf("failed to submit task to ants pool: %w", err)
		}

		obs.AddCounter(ctx, metricAsyncEnqueueTotal, 1,
			observabilityx.String("result", "submitted"),
			observabilityx.String("event_name", eventName(event)),
		)
		obs.RecordHistogram(ctx, metricAsyncEnqueueDurationMS, float64(time.Since(start).Milliseconds()),
			observabilityx.String("result", "submitted"),
			observabilityx.String("event_name", eventName(event)),
		)
		return nil
	}

	// Fallback to legacy channel-based queue
	if b.asyncQueue == nil {
		return b.Publish(ctx, event)
	}

	err := b.enqueueLegacyAsyncTask(publishTask{ctx: ctx, event: event, handlers: handlers})
	switch {
	case err == nil:
		obs.AddCounter(ctx, metricAsyncEnqueueTotal, 1,
			observabilityx.String("result", "enqueued"),
			observabilityx.String("event_name", eventName(event)),
		)
		obs.RecordHistogram(ctx, metricAsyncEnqueueDurationMS, float64(time.Since(start).Milliseconds()),
			observabilityx.String("result", "enqueued"),
			observabilityx.String("event_name", eventName(event)),
		)
		return nil
	case errors.Is(err, ErrBusClosed):
		span.RecordError(err)
		obs.AddCounter(ctx, metricAsyncEnqueueTotal, 1,
			observabilityx.String("result", "closed"),
			observabilityx.String("event_name", eventName(event)),
		)
		obs.RecordHistogram(ctx, metricAsyncEnqueueDurationMS, float64(time.Since(start).Milliseconds()),
			observabilityx.String("result", "closed"),
			observabilityx.String("event_name", eventName(event)),
		)
		return ErrBusClosed
	case errors.Is(err, errAsyncQueueUnavailable):
		return b.Publish(ctx, event)
	default:
		span.RecordError(ErrAsyncQueueFull)
		obs.AddCounter(ctx, metricAsyncEnqueueTotal, 1,
			observabilityx.String("result", "queue_full"),
			observabilityx.String("event_name", eventName(event)),
		)
		obs.RecordHistogram(ctx, metricAsyncEnqueueDurationMS, float64(time.Since(start).Milliseconds()),
			observabilityx.String("result", "queue_full"),
			observabilityx.String("event_name", eventName(event)),
		)
		return ErrAsyncQueueFull
	}
}

func (b *Bus) executeTask(task publishTask) {
	err := b.dispatch(task.ctx, task.event, task.handlers, "async")
	if err != nil && b.onAsyncErr != nil {
		b.onAsyncErr(task.ctx, task.event, err)
	} else if err != nil {
		b.logger.Warn("async dispatch failed",
			"event_name", eventName(task.event),
			"error", err.Error(),
		)
	}
	if err != nil {
		b.observabilitySafe().AddCounter(task.ctx, metricAsyncDispatchErrorTotal, 1,
			observabilityx.String("event_name", eventName(task.event)),
		)
	}
}

func (b *Bus) workerLoop() {
	defer b.workerWG.Done()
	for task := range b.asyncQueue {
		func(t publishTask) {
			defer b.dispatchWG.Done()
			b.executeTask(t)
		}(task)
	}
}
