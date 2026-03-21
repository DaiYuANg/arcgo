package dbx

import (
	"database/sql"
	"strings"

	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/samber/lo"
)

// OpenOption configures Open. Required: WithDriver, WithDSN, WithDialect.
// Use ApplyOptions to pass Option (WithLogger, WithHooks, WithDebug).
type OpenOption func(*openConfig) error

type openConfig struct {
	driver  string
	dsn     string
	dialect dialect.Dialect
	observe options
}

func defaultOpenConfig() openConfig {
	return openConfig{
		observe: defaultOptions(),
	}
}

// WithDriver sets the database driver name (e.g. "sqlite", "mysql", "postgres"). Required for Open.
func WithDriver(driver string) OpenOption {
	return func(c *openConfig) error {
		c.driver = strings.TrimSpace(driver)
		return nil
	}
}

// WithDSN sets the data source name. Required for Open.
func WithDSN(dsn string) OpenOption {
	return func(c *openConfig) error {
		c.dsn = strings.TrimSpace(dsn)
		return nil
	}
}

// WithDialect sets the dialect for query building. Required for Open.
func WithDialect(d dialect.Dialect) OpenOption {
	return func(c *openConfig) error {
		c.dialect = d
		return nil
	}
}

// ApplyOptions applies Option (WithLogger, WithHooks, WithDebug) to the DB created by Open.
func ApplyOptions(opts ...Option) OpenOption {
	return func(c *openConfig) error {
		lo.ForEach(lo.Filter(opts, func(opt Option, _ int) bool {
			return opt != nil
		}), func(opt Option, _ int) {
			opt(&c.observe)
		})
		return nil
	}
}

// Open creates a DB with connection managed internally. Requires WithDriver, WithDSN, WithDialect.
// Returns error if any required option is missing or invalid. Call db.Close() when done.
func Open(opts ...OpenOption) (*DB, error) {
	config := defaultOpenConfig()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&config); err != nil {
			return nil, err
		}
	}

	if config.driver == "" {
		return nil, ErrMissingDriver
	}
	if config.dsn == "" {
		return nil, ErrMissingDSN
	}
	if config.dialect == nil {
		return nil, ErrMissingDialect
	}

	raw, err := sql.Open(config.driver, config.dsn)
	if err != nil {
		return nil, err
	}

	return NewWithOptions(raw, config.dialect,
		WithLogger(config.observe.logger),
		WithHooks(config.observe.hooks...),
		WithDebug(config.observe.debug),
	), nil
}
