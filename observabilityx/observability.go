package observabilityx

import (
	"context"
	"log/slog"
)

// Attribute is a lightweight key/value pair used by observability APIs.
type Attribute struct {
	Key   string
	Value any
}

// String creates a string attribute.
func String(key, value string) Attribute {
	return Attribute{Key: key, Value: value}
}

// Bool creates a bool attribute.
func Bool(key string, value bool) Attribute {
	return Attribute{Key: key, Value: value}
}

// Int64 creates an int64 attribute.
func Int64(key string, value int64) Attribute {
	return Attribute{Key: key, Value: value}
}

// Float64 creates a float64 attribute.
func Float64(key string, value float64) Attribute {
	return Attribute{Key: key, Value: value}
}

// Any creates an attribute with arbitrary value.
func Any(key string, value any) Attribute {
	return Attribute{Key: key, Value: value}
}

// Span is a minimal tracing span contract.
type Span interface {
	End()
	RecordError(err error)
	SetAttributes(attrs ...Attribute)
}

// Counter records increasing int64 measurements.
type Counter interface {
	Add(ctx context.Context, value int64, attrs ...Attribute)
}

// UpDownCounter records signed int64 measurements.
type UpDownCounter interface {
	Add(ctx context.Context, value int64, attrs ...Attribute)
}

// Histogram records float64 measurements.
type Histogram interface {
	Record(ctx context.Context, value float64, attrs ...Attribute)
}

// Gauge records instantaneous float64 measurements.
type Gauge interface {
	Set(ctx context.Context, value float64, attrs ...Attribute)
}

// Observability is the shared observability facade used by arcgo packages.
//
// Implementations are expected to be safe for concurrent use.
type Observability interface {
	Logger() *slog.Logger
	StartSpan(ctx context.Context, name string, attrs ...Attribute) (context.Context, Span)
	Counter(spec CounterSpec) Counter
	UpDownCounter(spec UpDownCounterSpec) UpDownCounter
	Histogram(spec HistogramSpec) Histogram
	Gauge(spec GaugeSpec) Gauge
}

// Normalize returns a usable observability instance.
func Normalize(obs Observability, logger *slog.Logger) Observability {
	if obs != nil {
		return obs
	}
	return NopWithLogger(logger)
}

// NormalizeLogger returns default slog logger when input is nil.
func NormalizeLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}
