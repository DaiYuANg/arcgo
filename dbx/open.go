package dbx

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/DaiYuANg/arcgo/pkg/option"
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
		observe, err := applyOptions(opts...)
		if err != nil {
			return err
		}
		c.observe = observe
		return nil
	}
}

// Open creates a DB with connection managed internally. Requires WithDriver, WithDSN, WithDialect.
// Returns error if any required option is missing or invalid. Call db.Close() when done.
func Open(opts ...OpenOption) (*DB, error) {
	config := defaultOpenConfig()
	logRuntimeNodeWithLogger(config.observe.logger, config.observe.debug, "db.open.start", "options", len(opts))
	if err := option.ApplyErr(&config, opts...); err != nil {
		logRuntimeNodeWithLogger(config.observe.logger, config.observe.debug, "db.open.error", "stage", "apply_option", "error", err)
		return nil, fmt.Errorf("dbx: apply open options: %w", err)
	}
	logRuntimeNodeWithLogger(config.observe.logger, config.observe.debug,
		"db.open.configured",
		"driver", config.driver,
		"dialect", dialectName(config.dialect),
		"hooks", len(config.observe.hooks),
	)

	if config.driver == "" {
		logRuntimeNodeWithLogger(config.observe.logger, config.observe.debug, "db.open.error", "stage", "validate", "error", ErrMissingDriver)
		return nil, ErrMissingDriver
	}
	if config.dsn == "" {
		logRuntimeNodeWithLogger(config.observe.logger, config.observe.debug, "db.open.error", "stage", "validate", "error", ErrMissingDSN)
		return nil, ErrMissingDSN
	}
	if config.dialect == nil {
		logRuntimeNodeWithLogger(config.observe.logger, config.observe.debug, "db.open.error", "stage", "validate", "error", ErrMissingDialect)
		return nil, ErrMissingDialect
	}

	raw, err := sql.Open(config.driver, config.dsn)
	if err != nil {
		logRuntimeNodeWithLogger(config.observe.logger, config.observe.debug, "db.open.error", "stage", "sql_open", "error", err)
		return nil, wrapDBError("open database", err)
	}
	logRuntimeNodeWithLogger(config.observe.logger, config.observe.debug, "db.open.sql_opened", "driver", config.driver)

	dbOpts := []Option{
		WithLogger(config.observe.logger),
		WithHooks(config.observe.hooks...),
		WithDebug(config.observe.debug),
	}
	if config.observe.hasIDGenerator {
		dbOpts = append(dbOpts, WithIDGenerator(config.observe.idGenerator))
	}
	if config.observe.hasNodeID {
		dbOpts = append(dbOpts, WithNodeID(config.observe.nodeID))
	}
	db, err := NewWithOptions(raw, config.dialect, dbOpts...)
	if err != nil {
		logRuntimeNodeWithLogger(config.observe.logger, config.observe.debug, "db.open.error", "stage", "new_with_options", "error", err)
		return nil, err
	}
	logRuntimeNodeWithLogger(config.observe.logger, config.observe.debug, "db.open.done", "driver", config.driver, "dialect", dialectName(config.dialect))
	return db, nil
}
