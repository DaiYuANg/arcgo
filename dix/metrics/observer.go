package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/observabilityx"
)

// Option configures the dix metrics observer.
type Option func(*config)

type config struct {
	metricPrefix            string
	includeVersionAttribute bool
	includeHealthCheckName  bool
}

// WithMetricPrefix overrides the metric prefix used by emitted metrics.
func WithMetricPrefix(prefix string) Option {
	return func(cfg *config) {
		clean := strings.TrimSpace(prefix)
		if clean != "" {
			cfg.metricPrefix = clean
		}
	}
}

// WithVersionAttribute controls whether the app version is attached when available.
func WithVersionAttribute(enabled bool) Option {
	return func(cfg *config) {
		cfg.includeVersionAttribute = enabled
	}
}

// WithHealthCheckNameAttribute controls whether health check names are attached.
func WithHealthCheckNameAttribute(enabled bool) Option {
	return func(cfg *config) {
		cfg.includeHealthCheckName = enabled
	}
}

// NewObserver creates a dix.Observer that emits metrics through observabilityx.
func NewObserver(obs observabilityx.Observability, opts ...Option) dix.Observer {
	cfg := config{
		metricPrefix:            "dix",
		includeVersionAttribute: true,
		includeHealthCheckName:  true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return &observer{
		obs: observabilityx.Normalize(obs, nil),
		cfg: cfg,
	}
}

// WithObservability adapts an observability backend into a dix AppOption.
func WithObservability(obs observabilityx.Observability, opts ...Option) dix.AppOption {
	return dix.WithObserver(NewObserver(obs, opts...))
}

type observer struct {
	obs observabilityx.Observability
	cfg config
}

func (o *observer) OnBuild(ctx context.Context, event dix.BuildEvent) {
	attrs := o.withResultAttrs(event.Meta, event.Profile, event.Err)
	o.obs.AddCounter(ctx, o.metricName("build_total"), 1, attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("build_duration_ms"), durationMS(event.Duration), attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("build_modules"), float64(event.ModuleCount), attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("build_providers"), float64(event.ProviderCount), attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("build_hooks"), float64(event.HookCount), attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("build_setups"), float64(event.SetupCount), attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("build_invokes"), float64(event.InvokeCount), attrs...)
}

func (o *observer) OnStart(ctx context.Context, event dix.StartEvent) {
	attrs := o.withResultAttrs(event.Meta, event.Profile, event.Err)
	o.obs.AddCounter(ctx, o.metricName("start_total"), 1, attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("start_duration_ms"), durationMS(event.Duration), attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("start_registered_hooks"), float64(event.StartHookCount), attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("start_completed_hooks"), float64(event.StartedHookCount), attrs...)
	if event.RolledBack {
		o.obs.AddCounter(ctx, o.metricName("start_rollback_total"), 1, attrs...)
	}
}

func (o *observer) OnStop(ctx context.Context, event dix.StopEvent) {
	attrs := o.withResultAttrs(event.Meta, event.Profile, event.Err)
	o.obs.AddCounter(ctx, o.metricName("stop_total"), 1, attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("stop_duration_ms"), durationMS(event.Duration), attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("stop_registered_hooks"), float64(event.StopHookCount), attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("stop_shutdown_errors"), float64(event.ShutdownErrorCount), attrs...)
	if event.HookError {
		o.obs.AddCounter(ctx, o.metricName("stop_hook_error_total"), 1, attrs...)
	}
}

func (o *observer) OnHealthCheck(ctx context.Context, event dix.HealthCheckEvent) {
	attrs := o.commonAttrs(event.Meta, event.Profile)
	attrs = append(attrs,
		observabilityx.String("kind", string(event.Kind)),
		observabilityx.String("result", resultOf(event.Err)),
	)
	if o.cfg.includeHealthCheckName && event.Name != "" {
		attrs = append(attrs, observabilityx.String("check", event.Name))
	}
	o.obs.AddCounter(ctx, o.metricName("health_check_total"), 1, attrs...)
	o.obs.RecordHistogram(ctx, o.metricName("health_check_duration_ms"), durationMS(event.Duration), attrs...)
}

func (o *observer) OnStateTransition(ctx context.Context, event dix.StateTransitionEvent) {
	attrs := o.commonAttrs(event.Meta, event.Profile)
	attrs = append(attrs,
		observabilityx.String("from", event.From.String()),
		observabilityx.String("to", event.To.String()),
	)
	o.obs.AddCounter(ctx, o.metricName("state_transition_total"), 1, attrs...)
}

func (o *observer) commonAttrs(meta dix.AppMeta, profile dix.Profile) []observabilityx.Attribute {
	attrs := []observabilityx.Attribute{
		observabilityx.String("app", meta.Name),
		observabilityx.String("profile", string(profile)),
	}
	if o.cfg.includeVersionAttribute && strings.TrimSpace(meta.Version) != "" {
		attrs = append(attrs, observabilityx.String("version", meta.Version))
	}
	return attrs
}

func (o *observer) withResultAttrs(meta dix.AppMeta, profile dix.Profile, err error) []observabilityx.Attribute {
	attrs := o.commonAttrs(meta, profile)
	return append(attrs, observabilityx.String("result", resultOf(err)))
}

func (o *observer) metricName(suffix string) string {
	return o.cfg.metricPrefix + "_" + suffix
}

func resultOf(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

func durationMS(durationValue time.Duration) float64 {
	return float64(durationValue.Milliseconds())
}
