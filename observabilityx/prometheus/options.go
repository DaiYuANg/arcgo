package prometheus

import (
	"log/slog"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	prom "github.com/prometheus/client_golang/prometheus"
)

const defaultNamespace = "arcgo"

// Option configures Prometheus observability adapter.
type Option func(*config)

type config struct {
	logger    *slog.Logger
	namespace string
	register  prom.Registerer
	gatherer  prom.Gatherer
	buckets   collectionx.List[float64]
}

func defaultConfig() config {
	return config{
		logger:    slog.Default(),
		namespace: defaultNamespace,
		register:  prom.DefaultRegisterer,
		gatherer:  prom.DefaultGatherer,
		buckets:   collectionx.NewList(prom.DefBuckets...),
	}
}

// WithLogger sets logger used by this adapter.
func WithLogger(logger *slog.Logger) Option {
	return func(cfg *config) {
		cfg.logger = logger
	}
}

// WithNamespace sets namespace prefix for metric names.
func WithNamespace(namespace string) Option {
	return func(cfg *config) {
		cfg.namespace = strings.TrimSpace(namespace)
	}
}

// WithRegisterer sets custom metric registerer.
func WithRegisterer(registerer prom.Registerer) Option {
	return func(cfg *config) {
		if registerer != nil {
			cfg.register = registerer
		}
	}
}

// WithGatherer sets custom metric gatherer.
func WithGatherer(gatherer prom.Gatherer) Option {
	return func(cfg *config) {
		if gatherer != nil {
			cfg.gatherer = gatherer
		}
	}
}

// WithHistogramBuckets sets custom histogram buckets.
func WithHistogramBuckets(buckets collectionx.List[float64]) Option {
	return func(cfg *config) {
		if buckets != nil && !buckets.IsEmpty() {
			cfg.buckets = buckets
		}
	}
}
