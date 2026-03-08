package clientx

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/samber/lo"
)

type testObservability struct {
	counterCalls []counterCall
	histCalls    []histCall
}

type counterCall struct {
	name  string
	value int64
	attrs map[string]any
}

type histCall struct {
	name  string
	value float64
	attrs map[string]any
}

func (t *testObservability) Logger() *slog.Logger {
	return slog.Default()
}

func (t *testObservability) StartSpan(ctx context.Context, name string, attrs ...observabilityx.Attribute) (context.Context, observabilityx.Span) {
	_ = name
	_ = attrs
	return ctx, testSpan{}
}

func (t *testObservability) AddCounter(ctx context.Context, name string, value int64, attrs ...observabilityx.Attribute) {
	_ = ctx
	t.counterCalls = append(t.counterCalls, counterCall{
		name:  name,
		value: value,
		attrs: toAttrMap(attrs),
	})
}

func (t *testObservability) RecordHistogram(ctx context.Context, name string, value float64, attrs ...observabilityx.Attribute) {
	_ = ctx
	t.histCalls = append(t.histCalls, histCall{
		name:  name,
		value: value,
		attrs: toAttrMap(attrs),
	})
}

type testSpan struct{}

func (s testSpan) End() {}

func (s testSpan) RecordError(err error) {
	_ = err
}

func (s testSpan) SetAttributes(attrs ...observabilityx.Attribute) {
	_ = attrs
}

func TestObservabilityHookDefaultMetrics(t *testing.T) {
	obs := &testObservability{}
	h := NewObservabilityHook(obs)

	h.OnDial(DialEvent{
		Protocol: ProtocolTCP,
		Op:       "dial",
		Network:  "tcp",
		Addr:     "127.0.0.1:9000",
		Duration: 12 * time.Millisecond,
	})
	h.OnIO(IOEvent{
		Protocol: ProtocolTCP,
		Op:       "read",
		Addr:     "127.0.0.1:9000",
		Bytes:    20,
		Duration: 8 * time.Millisecond,
		Err:      WrapError(ProtocolTCP, "read", "127.0.0.1:9000", context.DeadlineExceeded),
	})

	assertCounterCall(t, obs.counterCalls, "clientx_dial_total", 1)
	assertCounterCall(t, obs.counterCalls, "clientx_io_total", 1)
	assertCounterCall(t, obs.counterCalls, "clientx_io_bytes_total", 20)
	assertHistogramCall(t, obs.histCalls, "clientx_dial_duration_ms")
	assertHistogramCall(t, obs.histCalls, "clientx_io_duration_ms")

	ioTotal, ok := lo.Find(obs.counterCalls, func(call counterCall) bool { return call.name == "clientx_io_total" })
	if !ok {
		t.Fatal("expected io_total call")
	}
	if got := ioTotal.attrs["result"]; got != "error" {
		t.Fatalf("expected result=error, got %v", got)
	}
	if got := ioTotal.attrs["error_kind"]; got != string(ErrorKindTimeout) {
		t.Fatalf("expected error_kind=%q, got %v", ErrorKindTimeout, got)
	}
}

func TestObservabilityHookWithOptions(t *testing.T) {
	obs := &testObservability{}
	h := NewObservabilityHook(
		obs,
		WithHookMetricPrefix("arc_client"),
		WithHookAddressAttribute(true),
	)

	h.OnDial(DialEvent{
		Protocol: ProtocolUDP,
		Op:       "dial",
		Network:  "udp",
		Addr:     "127.0.0.1:9001",
		Duration: 5 * time.Millisecond,
		Err:      errors.New("boom"),
	})

	call, ok := lo.Find(obs.counterCalls, func(c counterCall) bool { return c.name == "arc_client_dial_total" })
	if !ok {
		t.Fatal("expected arc_client_dial_total call")
	}
	if got := call.attrs["addr"]; got != "127.0.0.1:9001" {
		t.Fatalf("expected addr attr, got %v", got)
	}
	if got := call.attrs["result"]; got != "error" {
		t.Fatalf("expected result=error, got %v", got)
	}
}

func toAttrMap(attrs []observabilityx.Attribute) map[string]any {
	m := make(map[string]any, len(attrs))
	lo.ForEach(attrs, func(attr observabilityx.Attribute, _ int) {
		m[attr.Key] = attr.Value
	})
	return m
}

func assertCounterCall(t *testing.T, calls []counterCall, name string, value int64) {
	t.Helper()
	call, ok := lo.Find(calls, func(c counterCall) bool { return c.name == name })
	if !ok {
		t.Fatalf("expected counter call %q", name)
	}
	if call.value != value {
		t.Fatalf("counter %q expected value %d, got %d", name, value, call.value)
	}
}

func assertHistogramCall(t *testing.T, calls []histCall, name string) {
	t.Helper()
	_, ok := lo.Find(calls, func(c histCall) bool { return c.name == name })
	if !ok {
		t.Fatalf("expected histogram call %q", name)
	}
}
