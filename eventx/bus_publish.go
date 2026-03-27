package eventx

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/panjf2000/ants/v2"
)

// Publish dispatches one event synchronously to all matching subscribers.
func (b *Bus) Publish(ctx context.Context, event Event) error {
	if err := validatePublishRequest(b, event); err != nil {
		return err
	}
	ctx = normalizeContext(ctx)

	if !b.beginDispatch() {
		return ErrBusClosed
	}
	defer b.dispatchWG.Done()

	handlers := b.snapshotHandlersByEventType(reflect.TypeOf(event))
	b.logger.Debug("publish sync",
		"event_name", eventName(event),
		"handler_count", len(handlers),
	)

	return b.dispatch(ctx, event, handlers, "sync")
}

// PublishAsync enqueues one event for asynchronous dispatch.
func (b *Bus) PublishAsync(ctx context.Context, event Event) error {
	if err := validatePublishRequest(b, event); err != nil {
		return err
	}
	ctx = normalizeContext(ctx)

	eventLabel := eventName(event)
	obs := b.observabilitySafe()
	start := time.Now()
	ctx, span := obs.StartSpan(ctx, "eventx.publish.async.enqueue",
		observabilityx.String("event_name", eventLabel),
	)
	defer span.End()

	handlers := b.snapshotHandlersByEventType(reflect.TypeOf(event))
	b.logger.Debug("publish async requested",
		"event_name", eventLabel,
		"handler_count", len(handlers),
	)

	if err := b.asyncRuntimeUnavailable(); err != nil {
		return b.finishAsyncEnqueueError(ctx, obs, span, start, eventLabel, err, "unavailable")
	}
	if b.antsPool == nil {
		return b.Publish(ctx, event)
	}

	if err := b.submitAsyncTask(ctx, event, handlers); err != nil {
		return b.handleAsyncSubmitError(ctx, obs, span, start, eventLabel, err)
	}

	recordAsyncEnqueueMetrics(ctx, obs, start, eventLabel, "submitted")
	b.logger.Debug("publish async submitted",
		"event_name", eventLabel,
		"handler_count", len(handlers),
	)
	return nil
}

func (b *Bus) executeTask(task publishTask) {
	b.logger.Debug("async dispatch started",
		"event_name", eventName(task.event),
		"handler_count", len(task.handlers),
	)
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
	b.logger.Debug("async dispatch finished",
		"event_name", eventName(task.event),
		"handler_count", len(task.handlers),
		"has_error", err != nil,
	)
}

func validatePublishRequest(b *Bus, event Event) error {
	if b == nil {
		return ErrNilBus
	}
	if event == nil {
		return ErrNilEvent
	}
	return nil
}

func (b *Bus) asyncRuntimeUnavailable() error {
	if b == nil || b.initErr == nil {
		return nil
	}
	return errors.Join(ErrAsyncRuntimeUnavailable, b.initErr)
}

func (b *Bus) finishAsyncEnqueueError(
	ctx context.Context,
	obs observabilityx.Observability,
	span observabilityx.Span,
	start time.Time,
	event string,
	err error,
	result string,
) error {
	b.logger.Debug("publish async unavailable",
		"event_name", event,
		"error", err,
	)
	span.RecordError(err)
	recordAsyncEnqueueMetrics(ctx, obs, start, event, result)
	return err
}

func (b *Bus) submitAsyncTask(ctx context.Context, event Event, handlers []HandlerFunc) error {
	if !b.beginDispatch() {
		return ErrBusClosed
	}

	task := publishTask{
		ctx:      ctx,
		event:    event,
		handlers: handlers,
	}
	if err := b.antsPool.Submit(func() {
		defer b.dispatchWG.Done()
		b.executeTask(task)
	}); err != nil {
		b.dispatchWG.Done()
		if errors.Is(err, ants.ErrPoolClosed) {
			return ErrBusClosed
		}
		return fmt.Errorf("eventx: submit async task: %w", err)
	}
	return nil
}

func (b *Bus) handleAsyncSubmitError(
	ctx context.Context,
	obs observabilityx.Observability,
	span observabilityx.Span,
	start time.Time,
	event string,
	err error,
) error {
	result := "pool_error"
	if errors.Is(err, ErrBusClosed) {
		result = "closed"
	}

	b.logger.Debug("publish async submit failed",
		"event_name", event,
		"error", err,
	)
	span.RecordError(err)
	recordAsyncEnqueueMetrics(ctx, obs, start, event, result)
	return err
}
