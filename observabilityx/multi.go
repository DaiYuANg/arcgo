package observabilityx

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

// Multi combines multiple observability backends into one.
//
// Use this to send telemetry to more than one backend (for example OTel + Prometheus).
func Multi(backends ...Observability) Observability {
	filtered := lo.Filter(backends, func(backend Observability, _ int) bool {
		return backend != nil
	})
	if len(filtered) == 0 {
		return Nop()
	}

	logger := filtered[0].Logger()
	if logger == nil {
		logger = slog.Default()
	}

	return &multiObservability{
		backends: filtered,
		logger:   logger,
	}
}

type multiObservability struct {
	backends []Observability
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

	nextCtx, firstSpan := m.backends[0].StartSpan(ctx, name, attrs...)
	spans := collectionx.NewListWithCapacity[Span](len(m.backends))
	if firstSpan != nil {
		spans.Add(firstSpan)
	}
	spans.MergeSlice(lo.FilterMap(m.backends[1:], func(backend Observability, _ int) (Span, bool) {
		_, span := backend.StartSpan(nextCtx, name, attrs...)
		return span, span != nil
	}))
	if spans.Len() == 0 {
		return nextCtx, nopSpan{}
	}
	return nextCtx, multiSpan{spans: spans.Values()}
}

func (m *multiObservability) AddCounter(ctx context.Context, name string, value int64, attrs ...Attribute) {
	lo.ForEach(m.backends, func(backend Observability, _ int) {
		backend.AddCounter(ctx, name, value, attrs...)
	})
}

func (m *multiObservability) RecordHistogram(ctx context.Context, name string, value float64, attrs ...Attribute) {
	lo.ForEach(m.backends, func(backend Observability, _ int) {
		backend.RecordHistogram(ctx, name, value, attrs...)
	})
}

type multiSpan struct {
	spans []Span
}

func (s multiSpan) End() {
	lo.ForEach(s.spans, func(span Span, _ int) {
		span.End()
	})
}

func (s multiSpan) RecordError(err error) {
	if err == nil {
		return
	}
	lo.ForEach(s.spans, func(span Span, _ int) {
		span.RecordError(err)
	})
}

func (s multiSpan) SetAttributes(attrs ...Attribute) {
	lo.ForEach(s.spans, func(span Span, _ int) {
		span.SetAttributes(attrs...)
	})
}
