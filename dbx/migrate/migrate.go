package migrate

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/samber/lo"
)

type Kind string

type Direction string

const (
	KindGo         Kind = "go"
	KindSQL        Kind = "sql"
	KindRepeatable Kind = "repeatable"
)

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

var ErrInvalidVersionedFilename = errors.New("dbx/migrate: invalid versioned filename")

type Migration interface {
	Version() string
	Description() string
	Up(ctx context.Context, tx *sql.Tx) error
	Down(ctx context.Context, tx *sql.Tx) error
}

type GoMigration struct {
	version     string
	description string
	up          func(context.Context, *sql.Tx) error
	down        func(context.Context, *sql.Tx) error
}

func NewGoMigration(version, description string, up, down func(context.Context, *sql.Tx) error) GoMigration {
	return GoMigration{version: version, description: description, up: up, down: down}
}

func (m GoMigration) Version() string     { return m.version }
func (m GoMigration) Description() string { return m.description }

func (m GoMigration) Up(ctx context.Context, tx *sql.Tx) error {
	if m.up == nil {
		return nil
	}
	return m.up(ctx, tx)
}

func (m GoMigration) Down(ctx context.Context, tx *sql.Tx) error {
	if m.down == nil {
		return nil
	}
	return m.down(ctx, tx)
}

type VersionedFile struct {
	Version     string
	Description string
	Kind        Kind
	Direction   Direction
	Path        string
	Filename    string
}

type SQLMigration struct {
	Version     string
	Description string
	UpPath      string
	DownPath    string
	Repeatable  bool
}

type FileSource struct {
	FS  fs.FS
	Dir string
}

type RunnerOptions struct {
	HistoryTable    string
	AllowOutOfOrder bool
	ValidateHash    bool
}

type Runner struct {
	db      *sql.DB
	dialect dialect.Dialect
	options RunnerOptions
}

type AppliedRecord struct {
	Version     string
	Description string
	Kind        Kind
	AppliedAt   time.Time
	Checksum    string
	Success     bool
}

func NewRunner(db *sql.DB, d dialect.Dialect, opts RunnerOptions) *Runner {
	if opts.HistoryTable == "" {
		opts.HistoryTable = "schema_history"
	}
	return &Runner{db: db, dialect: d, options: opts}
}

func (r *Runner) DB() *sql.DB {
	return r.db
}

func (r *Runner) Dialect() dialect.Dialect {
	return r.dialect
}

func (r *Runner) Options() RunnerOptions {
	return r.options
}

var versionedFilePattern = regexp.MustCompile(`^(?P<prefix>V|U|R)(?P<version>[0-9A-Za-z_.-]*)__(?P<description>.+)\.sql$`)

func ParseVersionedFilename(name string) (VersionedFile, error) {
	base := filepath.Base(name)
	match := versionedFilePattern.FindStringSubmatch(base)
	if match == nil {
		return VersionedFile{}, ErrInvalidVersionedFilename
	}

	file := VersionedFile{
		Filename: base,
		Path:     name,
	}

	switch match[1] {
	case "V":
		file.Kind = KindSQL
		file.Direction = DirectionUp
	case "U":
		file.Kind = KindSQL
		file.Direction = DirectionDown
	case "R":
		file.Kind = KindRepeatable
		file.Direction = DirectionUp
	}

	file.Version = match[2]
	file.Description = strings.ReplaceAll(match[3], "_", " ")
	return file, nil
}

func (s FileSource) List() ([]SQLMigration, error) {
	entries, err := fs.ReadDir(s.FS, s.Dir)
	if err != nil {
		return nil, err
	}

	items := collectionx.NewMapWithCapacity[string, *SQLMigration](len(entries))
	for _, entry := range lo.Filter(entries, func(e fs.DirEntry, _ int) bool { return !e.IsDir() }) {
		// Validate path before parsing to reject path traversal in any filename
		fullPath, err := safeJoinPath(s.Dir, entry.Name())
		if err != nil {
			return nil, err
		}

		parsed, err := ParseVersionedFilename(entry.Name())
		if err != nil {
			continue
		}

		key := parsed.Version + ":" + parsed.Description
		migration, exists := items.Get(key)
		if !exists {
			migration = &SQLMigration{
				Version:     parsed.Version,
				Description: parsed.Description,
				Repeatable:  parsed.Kind == KindRepeatable,
			}
			items.Set(key, migration)
		}

		fullPath = filepath.ToSlash(fullPath)
		if parsed.Direction == DirectionUp {
			migration.UpPath = fullPath
		} else {
			migration.DownPath = fullPath
		}
	}

	result := collectionx.NewListWithCapacity[SQLMigration](items.Len())
	for _, migration := range items.Values() {
		result.Add(*migration)
	}

	itemsSlice := result.Values()
	sort.Slice(itemsSlice, func(i, j int) bool {
		if itemsSlice[i].Repeatable != itemsSlice[j].Repeatable {
			return !itemsSlice[i].Repeatable
		}
		if itemsSlice[i].Version != itemsSlice[j].Version {
			return itemsSlice[i].Version < itemsSlice[j].Version
		}
		return itemsSlice[i].Description < itemsSlice[j].Description
	})
	return itemsSlice, nil
}

// safeJoinPath joins base and name, returning an error if the result escapes base (path traversal).
func safeJoinPath(base, name string) (string, error) {
	base = filepath.Clean(base)
	path := filepath.Clean(filepath.Join(base, name))
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", errors.New("path traversal not allowed: " + name)
	}
	return path, nil
}
