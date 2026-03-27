package migrate

import (
	"fmt"
	"io/fs"
	"strings"
)

type loadedSQLMigration struct {
	SQLMigration
	kind     Kind
	upSQL    string
	downSQL  string
	checksum string
}

func loadSQLMigrations(source FileSource) ([]loadedSQLMigration, error) {
	items, err := source.List()
	if err != nil {
		return nil, fmt.Errorf("dbx/migrate: list sql migrations: %w", err)
	}

	loaded := make([]loadedSQLMigration, 0, len(items))
	for i := range items {
		migration := items[i]
		if migration.UpPath == "" {
			continue
		}
		item, loadErr := loadSQLMigration(source.FS, migration)
		if loadErr != nil {
			return nil, loadErr
		}
		loaded = append(loaded, item)
	}
	return loaded, nil
}

func loadSQLMigration(fsys fs.FS, migration SQLMigration) (loadedSQLMigration, error) {
	upSQL, err := readSQLFile(fsys, migration.UpPath)
	if err != nil {
		return loadedSQLMigration{}, err
	}

	downSQL := ""
	if migration.DownPath != "" {
		downSQL, err = readSQLFile(fsys, migration.DownPath)
		if err != nil {
			return loadedSQLMigration{}, err
		}
	}

	return loadedSQLMigration{
		SQLMigration: migration,
		kind:         kindForSQLMigration(migration),
		upSQL:        upSQL,
		downSQL:      downSQL,
		checksum:     checksumSQLMigration(migration, upSQL, downSQL),
	}, nil
}

func readSQLFile(fsys fs.FS, path string) (string, error) {
	bytes, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", fmt.Errorf("dbx/migrate: read sql file %q: %w", path, err)
	}
	return strings.TrimSpace(string(bytes)), nil
}

func kindForSQLMigration(migration SQLMigration) Kind {
	if migration.Repeatable {
		return KindRepeatable
	}
	return KindSQL
}
