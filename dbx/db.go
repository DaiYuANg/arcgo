package dbx

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"

	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type DB struct {
	raw         *sql.DB
	dialect     dialect.Dialect
	observe     runtimeObserver
	relation    *relationRuntime
	idGenerator IDGenerator
	nodeID      uint16
}

func New(raw *sql.DB, d dialect.Dialect) *DB {
	db, err := NewWithOptions(raw, d)
	if err != nil {
		panic(err)
	}
	return db
}

func NewWithOptions(raw *sql.DB, d dialect.Dialect, opts ...Option) (*DB, error) {
	config, err := applyOptions(opts...)
	if err != nil {
		return nil, err
	}
	logRuntimeNodeWithLogger(config.logger, config.debug,
		"db.new.start",
		"has_sql_db", raw != nil,
		"dialect", dialectName(d),
		"hooks", len(config.hooks),
		"has_id_generator", config.idGenerator != nil,
		"node_id", config.nodeID,
	)
	idGenerator := config.idGenerator
	if idGenerator == nil {
		idGenerator, err = NewSnowflakeGenerator(config.nodeID)
		if err != nil {
			logRuntimeNodeWithLogger(config.logger, config.debug, "db.new.error", "stage", "id_generator", "error", err)
			return nil, err
		}
	}
	db := &DB{
		raw:         raw,
		dialect:     d,
		observe:     newRuntimeObserver(config),
		relation:    newRelationRuntime(),
		idGenerator: idGenerator,
		nodeID:      config.nodeID,
	}
	logRuntimeNodeWithLogger(config.logger, config.debug,
		"db.new.done",
		"dialect", dialectName(d),
		"hooks", len(config.hooks),
		"node_id", config.nodeID,
	)
	return db, nil
}

func MustNewWithOptions(raw *sql.DB, d dialect.Dialect, opts ...Option) *DB {
	db, err := NewWithOptions(raw, d, opts...)
	if err != nil {
		panic(err)
	}
	return db
}

func (db *DB) SQLDB() *sql.DB {
	return db.raw
}

func (db *DB) Dialect() dialect.Dialect {
	return db.dialect
}

func (db *DB) WithSQLDB(raw *sql.DB) *DB {
	return &DB{
		raw:         raw,
		dialect:     db.dialect,
		observe:     db.observe,
		relation:    db.relation,
		idGenerator: db.idGenerator,
		nodeID:      db.nodeID,
	}
}

// RelationRuntime returns the relation load runtime (cache and pools) for this DB.
// Used by LoadBelongsTo, LoadHasMany, LoadManyToMany.
func (db *DB) RelationRuntime() *relationRuntime {
	if db == nil || db.relation == nil {
		return defaultRelationRuntime
	}
	return db.relation
}

func (db *DB) Logger() *slog.Logger {
	return db.observe.logger
}

func (db *DB) Hooks() []Hook {
	return slices.Clone(db.observe.hooks)
}

func (db *DB) Debug() bool {
	return db.observe.debug
}

func (db *DB) IDGenerator() IDGenerator {
	if db == nil {
		return nil
	}
	return db.idGenerator
}

func (db *DB) NodeID() uint16 {
	if db == nil {
		return 0
	}
	return db.nodeID
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.queryContext(ctx, "", query, args...)
}

func (db *DB) queryContext(ctx context.Context, statement string, query string, args ...any) (*sql.Rows, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	if db.raw == nil {
		return nil, ErrNilSQLDB
	}
	ctx, event, err := db.observe.before(ctx, HookEvent{
		Operation: OperationQuery,
		Statement: statement,
		SQL:       query,
		Args:      args,
	})
	if err != nil {
		db.observe.after(ctx, event)
		return nil, err
	}
	rows, queryErr := db.raw.QueryContext(ctx, query, args...)
	event.Err = queryErr
	db.observe.after(ctx, event)
	return rows, queryErr
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.execContext(ctx, "", query, args...)
}

func (db *DB) execContext(ctx context.Context, statement string, query string, args ...any) (sql.Result, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	if db.raw == nil {
		return nil, ErrNilSQLDB
	}
	ctx, event, err := db.observe.before(ctx, HookEvent{
		Operation: OperationExec,
		Statement: statement,
		SQL:       query,
		Args:      args,
	})
	if err != nil {
		db.observe.after(ctx, event)
		return nil, err
	}
	result, execErr := db.raw.ExecContext(ctx, query, args...)
	if execErr == nil && result != nil {
		if rowsAffected, rowsErr := result.RowsAffected(); rowsErr == nil {
			event.RowsAffected = rowsAffected
			event.HasRowsAffected = true
		}
	}
	event.Err = execErr
	db.observe.after(ctx, event)
	return result, execErr
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	if db == nil {
		return errorRow(ErrNilDB)
	}
	if db.raw == nil {
		return errorRow(ErrNilSQLDB)
	}
	ctx, event, err := db.observe.before(ctx, HookEvent{Operation: OperationQueryRow, SQL: query, Args: args})
	if err != nil {
		db.observe.after(ctx, event)
		return errorRow(err)
	}
	rows, queryErr := db.raw.QueryContext(ctx, query, args...)
	if queryErr != nil {
		event.Err = queryErr
		db.observe.after(ctx, event)
		return errorRow(queryErr)
	}
	return observedRow(ctx, db.observe, event, rows)
}

func (db *DB) Bound(sql string, args ...any) BoundQuery {
	return BoundQuery{SQL: sql, Args: slices.Clone(args)}
}

func (db *DB) QueryBoundContext(ctx context.Context, bound BoundQuery) (*sql.Rows, error) {
	return db.queryContext(ctx, bound.Name, bound.SQL, bound.Args...)
}

func (db *DB) ExecBoundContext(ctx context.Context, bound BoundQuery) (sql.Result, error) {
	return db.execContext(ctx, bound.Name, bound.SQL, bound.Args...)
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	if db.raw == nil {
		return nil, ErrNilSQLDB
	}
	ctx, event, err := db.observe.before(ctx, HookEvent{Operation: OperationBeginTx})
	if err != nil {
		db.observe.after(ctx, event)
		return nil, err
	}
	tx, err := db.raw.BeginTx(ctx, opts)
	if err != nil {
		event.Err = err
		db.observe.after(ctx, event)
		return nil, err
	}
	db.observe.after(ctx, event)
	return &Tx{raw: tx, dialect: db.dialect, observe: db.observe, relation: db.relation, idGenerator: db.idGenerator, nodeID: db.nodeID}, nil
}

func (db *DB) WithTx(tx *sql.Tx) *Tx {
	if tx == nil {
		return nil
	}
	return &Tx{raw: tx, dialect: db.dialect, observe: db.observe, relation: db.relation, idGenerator: db.idGenerator, nodeID: db.nodeID}
}

func (db *DB) SQL() *SQLExecutor {
	return &SQLExecutor{session: db}
}

// Close closes the underlying database connection. Call when the DB is no longer needed.
// Safe to call if raw is nil. After Close, the DB should not be used for execution.
func (db *DB) Close() error {
	if db == nil || db.raw == nil {
		return nil
	}
	logRuntimeNode(db, "db.close.start")
	err := db.raw.Close()
	if err != nil {
		logRuntimeNode(db, "db.close.error", "error", err)
		return err
	}
	logRuntimeNode(db, "db.close.done")
	return nil
}

func dialectName(d dialect.Dialect) string {
	if d == nil {
		return ""
	}
	return d.Name()
}
