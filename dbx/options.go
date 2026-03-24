package dbx

import (
	"log/slog"

	"github.com/samber/lo"
)

// Option configures a DB instance. Options are composable; later options override earlier ones.
type Option func(*options)

type options struct {
	logger         *slog.Logger
	hooks          []Hook
	debug          bool
	idGenerator    IDGenerator
	nodeID         uint16
	hasIDGenerator bool
	hasNodeID      bool
}

func defaultOptions() options {
	return options{
		logger: slog.Default(),
		hooks:  make([]Hook, 0, 4),
		debug:  false,
		nodeID: ResolveNodeIDFromHostName(),
	}
}

// DefaultOptions returns no options; use when you want explicit defaults (logger=slog.Default, debug=false, no hooks).
func DefaultOptions() []Option {
	return nil
}

// ProductionOptions returns options suitable for production: debug off, no extra hooks.
// Combine with WithLogger for custom logging. Same as defaults; use for explicitness.
func ProductionOptions() []Option {
	return []Option{WithDebug(false)}
}

// TestOptions returns options for tests: debug on (SQL logged). Combine with WithLogger, WithHooks as needed.
func TestOptions() []Option {
	return []Option{WithDebug(true)}
}

// WithLogger sets the logger for operation events. Default: slog.Default().
// When debug is false, only errors are logged; when true, all operations are logged at Debug level.
func WithLogger(logger *slog.Logger) Option {
	return func(opts *options) {
		if logger != nil {
			opts.logger = logger
		}
	}
}

// WithHooks appends hooks that run before/after each operation (query, exec, begin/commit/rollback, etc.).
// Hooks are additive; pass multiple or call WithHooks multiple times to combine.
func WithHooks(hooks ...Hook) Option {
	return func(opts *options) {
		opts.hooks = append(opts.hooks, lo.Filter(hooks, func(hook Hook, _ int) bool {
			return hook != nil
		})...)
	}
}

// WithDebug enables SQL logging for all operations when true. Default: false.
// When false, only errors are logged. Use in development or tests to inspect queries.
func WithDebug(enabled bool) Option {
	return func(opts *options) {
		opts.debug = enabled
	}
}

// WithIDGenerator sets the DB-scoped ID generator used by mapper insert assignment helpers.
// Mutually exclusive with WithNodeID.
func WithIDGenerator(generator IDGenerator) Option {
	return func(opts *options) {
		if generator == nil {
			return
		}
		opts.idGenerator = generator
		opts.hasIDGenerator = true
	}
}

// WithNodeID sets the DB node id used by the default Snowflake generator.
// Mutually exclusive with WithIDGenerator.
func WithNodeID(nodeID uint16) Option {
	return func(opts *options) {
		opts.nodeID = nodeID
		opts.hasNodeID = true
	}
}

func applyOptions(opts ...Option) (options, error) {
	config := defaultOptions()
	lo.ForEach(lo.Filter(opts, func(opt Option, _ int) bool {
		return opt != nil
	}), func(opt Option, _ int) {
		opt(&config)
	})
	if config.hasIDGenerator && config.hasNodeID {
		return options{}, ErrIDGeneratorNodeIDConflict
	}
	if config.hasNodeID {
		if config.nodeID < MinNodeID || config.nodeID > MaxNodeID {
			return options{}, &NodeIDOutOfRangeError{NodeID: config.nodeID, Min: MinNodeID, Max: MaxNodeID}
		}
	}
	return config, nil
}
