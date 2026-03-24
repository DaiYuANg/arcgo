package shared

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/dbx"
	sqlitedialect "github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
	"github.com/DaiYuANg/arcgo/logx"
	_ "modernc.org/sqlite"
)

func NewLogger() *slog.Logger {
	return logx.MustNew(
		logx.WithConsole(true),
		logx.WithLevel(slog.LevelDebug),
	)
}

// OpenSQLite opens a SQLite DB with connection managed by dbx. Returns (db, closeFn, err).
// Call closeFn() or db.Close() when done.
func OpenSQLite(name string, opts ...dbx.Option) (*dbx.DB, func() error, error) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
	db, err := dbx.Open(
		dbx.WithDriver("sqlite"),
		dbx.WithDSN(dsn),
		dbx.WithDialect(sqlitedialect.Dialect{}),
		dbx.ApplyOptions(opts...),
	)
	if err != nil {
		return nil, nil, err
	}
	if _, err := db.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return db, db.Close, nil
}

// OpenSQLiteRaw returns dbx wrapping an existing *sql.DB. Caller owns raw and must close it.
func OpenSQLiteRaw(name string, opts ...dbx.Option) (*dbx.DB, func() error, error) {
	raw, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", name))
	if err != nil {
		return nil, nil, err
	}
	if err := raw.Ping(); err != nil {
		_ = raw.Close()
		return nil, nil, err
	}
	if _, err := raw.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = raw.Close()
		return nil, nil, err
	}
	db, err := dbx.NewWithOptions(raw, sqlitedialect.Dialect{}, opts...)
	if err != nil {
		_ = raw.Close()
		return nil, nil, err
	}
	return db, raw.Close, nil
}
