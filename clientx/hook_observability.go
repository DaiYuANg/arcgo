package clientx

import (
	"context"
	"strings"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/samber/lo"
)

type ObservabilityHookOption func(*observabilityHookConfig)

type observabilityHookConfig struct {
	metricPrefix        string
	includeAddressAttrs bool
}

func WithHookMetricPrefix(prefix string) ObservabilityHookOption {
	return func(cfg *observabilityHookConfig) {
		clean := strings.TrimSpace(prefix)
		if clean != "" {
			cfg.metricPrefix = clean
		}
	}
}

func WithHookAddressAttribute(enabled bool) ObservabilityHookOption {
	return func(cfg *observabilityHookConfig) {
		cfg.includeAddressAttrs = enabled
	}
}

func NewObservabilityHook(obs observabilityx.Observability, opts ...ObservabilityHookOption) Hook {
	cfg := observabilityHookConfig{
		metricPrefix: "clientx",
	}
	lo.ForEach(opts, func(opt ObservabilityHookOption, _ int) {
		if opt != nil {
			opt(&cfg)
		}
	})

	return &observabilityHook{
		obs: observabilityx.Normalize(obs, nil),
		cfg: cfg,
	}
}

type observabilityHook struct {
	obs observabilityx.Observability
	cfg observabilityHookConfig
}

func (h *observabilityHook) OnDial(event DialEvent) {
	attrs := []observabilityx.Attribute{
		observabilityx.String("protocol", string(event.Protocol)),
		observabilityx.String("op", event.Op),
		observabilityx.String("network", event.Network),
		observabilityx.String("result", resultOf(event.Err)),
	}
	if h.cfg.includeAddressAttrs && event.Addr != "" {
		attrs = append(attrs, observabilityx.String("addr", event.Addr))
	}
	if event.Err != nil {
		attrs = append(attrs, observabilityx.String("error_kind", string(KindOf(event.Err))))
	}

	ctx := context.Background()
	h.obs.AddCounter(ctx, h.metricName("dial_total"), 1, attrs...)
	h.obs.RecordHistogram(ctx, h.metricName("dial_duration_ms"), float64(event.Duration.Milliseconds()), attrs...)
}

func (h *observabilityHook) OnIO(event IOEvent) {
	attrs := []observabilityx.Attribute{
		observabilityx.String("protocol", string(event.Protocol)),
		observabilityx.String("op", event.Op),
		observabilityx.String("result", resultOf(event.Err)),
	}
	if h.cfg.includeAddressAttrs && event.Addr != "" {
		attrs = append(attrs, observabilityx.String("addr", event.Addr))
	}
	if event.Err != nil {
		attrs = append(attrs, observabilityx.String("error_kind", string(KindOf(event.Err))))
	}

	ctx := context.Background()
	h.obs.AddCounter(ctx, h.metricName("io_total"), 1, attrs...)
	h.obs.RecordHistogram(ctx, h.metricName("io_duration_ms"), float64(event.Duration.Milliseconds()), attrs...)
	if event.Bytes > 0 {
		h.obs.AddCounter(ctx, h.metricName("io_bytes_total"), int64(event.Bytes), attrs...)
	}
}

func (h *observabilityHook) metricName(suffix string) string {
	return h.cfg.metricPrefix + "_" + suffix
}

func resultOf(err error) string {
	return lo.Ternary(err == nil, "ok", "error")
}
