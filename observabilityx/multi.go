package observabilityx

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
)

// Multi combines multiple observability backends into one.
//
// Use this to send telemetry to more than one backend (for example OTel + Prometheus).
func Multi(backends ...Observability) Observability {
	filtered := collectionx.NewList(backends...).Reject(func(_ int, backend Observability) bool {
		return backend == nil
	})
	if filtered.IsEmpty() {
		return Nop()
	}

	firstBackend, _ := filtered.GetFirst()
	logger := firstBackend.Logger()
	if logger == nil {
		logger = slog.Default()
	}

	return &multiObservability{
		backends: filtered,
		logger:   logger,
	}
}

type multiObservability struct {
	backends collectionx.List[Observability]
	logger   *slog.Logger
}

func (m *multiObservability) Logger() *slog.Logger {
	return NormalizeLogger(m.logger)
}

func (m *multiObservability) StartSpan(
	ctx context.Context,
	name string,
	attrs ...Attribute,
) (context.Context, Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	firstBackend, _ := m.backends.GetFirst()
	nextCtx, firstSpan := firstBackend.StartSpan(ctx, name, attrs...)
	spans := collectionx.NewListWithCapacity[Span](m.backends.Len())
	if firstSpan != nil {
		spans.Add(firstSpan)
	}
	m.backends.Drop(1).Range(func(_ int, backend Observability) bool {
		_, span := backend.StartSpan(nextCtx, name, attrs...)
		if span != nil {
			spans.Add(span)
		}
		return true
	})
	if spans.Len() == 0 {
		return nextCtx, nopSpan{}
	}
	return nextCtx, multiSpan{spans: spans}
}

func (m *multiObservability) AddCounter(ctx context.Context, name string, value int64, attrs ...Attribute) {
	m.backends.Each(func(_ int, backend Observability) {
		backend.AddCounter(ctx, name, value, attrs...)
	})
}

func (m *multiObservability) RecordHistogram(ctx context.Context, name string, value float64, attrs ...Attribute) {
	m.backends.Each(func(_ int, backend Observability) {
		backend.RecordHistogram(ctx, name, value, attrs...)
	})
}

type multiSpan struct {
	spans collectionx.List[Span]
}

func (s multiSpan) End() {
	s.spans.Each(func(_ int, span Span) {
		span.End()
	})
}

func (s multiSpan) RecordError(err error) {
	if err == nil {
		return
	}
	s.spans.Each(func(_ int, span Span) {
		span.RecordError(err)
	})
}

func (s multiSpan) SetAttributes(attrs ...Attribute) {
	s.spans.Each(func(_ int, span Span) {
		span.SetAttributes(attrs...)
	})
}
