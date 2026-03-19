package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

type RunReport struct {
	Applied []AppliedRecord
}

func (r *Runner) EnsureHistory(ctx context.Context) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}
	_, err := r.db.ExecContext(ctx, historyTableDDL(r.dialect, r.options.HistoryTable))
	return err
}

func (r *Runner) Applied(ctx context.Context) ([]AppliedRecord, error) {
	if r == nil || r.db == nil {
		return nil, sql.ErrConnDone
	}
	if err := r.EnsureHistory(ctx); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, appliedRecordsSQL(r.dialect, r.options.HistoryTable))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AppliedRecord, 0, 8)
	for rows.Next() {
		var (
			record      AppliedRecord
			kind        string
			appliedAt   string
			successFlag bool
		)
		if err := rows.Scan(&record.Version, &record.Description, &kind, &appliedAt, &record.Checksum, &successFlag); err != nil {
			return nil, err
		}
		parsedTime, err := time.Parse(timeLayout, appliedAt)
		if err != nil {
			return nil, fmt.Errorf("dbx/migrate: parse applied_at: %w", err)
		}
		record.Kind = Kind(kind)
		record.AppliedAt = parsedTime
		record.Success = successFlag
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Runner) PendingGo(ctx context.Context, migrations ...Migration) ([]Migration, error) {
	applied, err := r.Applied(ctx)
	if err != nil {
		return nil, err
	}
	indexed := indexAppliedRecords(applied)
	pending := make([]Migration, 0, len(migrations))
	sorted := append([]Migration(nil), migrations...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Version() != sorted[j].Version() {
			return sorted[i].Version() < sorted[j].Version()
		}
		return sorted[i].Description() < sorted[j].Description()
	})

	for _, migration := range sorted {
		key := appliedRecordKey(KindGo, migration.Version(), migration.Description())
		checksum := checksumGoMigration(migration)
		if record, ok := indexed[key]; ok {
			if r.options.ValidateHash && record.Checksum != checksum {
				return nil, fmt.Errorf("dbx/migrate: go migration checksum mismatch for version %s", migration.Version())
			}
			continue
		}
		if !r.options.AllowOutOfOrder && hasAppliedNewerVersion(applied, migration.Version()) {
			return nil, fmt.Errorf("dbx/migrate: out-of-order go migration version %s", migration.Version())
		}
		pending = append(pending, migration)
	}
	return pending, nil
}

func (r *Runner) PendingSQL(ctx context.Context, source FileSource) ([]SQLMigration, error) {
	loaded, err := loadSQLMigrations(source)
	if err != nil {
		return nil, err
	}
	applied, err := r.Applied(ctx)
	if err != nil {
		return nil, err
	}
	indexed := indexAppliedRecords(applied)
	pending := make([]SQLMigration, 0, len(loaded))

	for _, migration := range loaded {
		key := appliedRecordKey(migration.kind, migration.Version, migration.Description)
		record, ok := indexed[key]
		if ok {
			if migration.kind == KindRepeatable {
				if record.Checksum != migration.checksum {
					pending = append(pending, migration.SQLMigration)
				}
				continue
			}
			if r.options.ValidateHash && record.Checksum != migration.checksum {
				return nil, fmt.Errorf("dbx/migrate: sql migration checksum mismatch for version %s", migration.Version)
			}
			continue
		}
		if !migration.Repeatable && !r.options.AllowOutOfOrder && hasAppliedNewerVersion(applied, migration.Version) {
			return nil, fmt.Errorf("dbx/migrate: out-of-order sql migration version %s", migration.Version)
		}
		pending = append(pending, migration.SQLMigration)
	}
	return pending, nil
}

func (r *Runner) UpGo(ctx context.Context, migrations ...Migration) (RunReport, error) {
	pending, err := r.PendingGo(ctx, migrations...)
	if err != nil {
		return RunReport{}, err
	}

	report := RunReport{Applied: make([]AppliedRecord, 0, len(pending))}
	for _, migration := range pending {
		applied, err := r.applyGoMigration(ctx, migration)
		if err != nil {
			return report, err
		}
		report.Applied = append(report.Applied, applied)
	}
	return report, nil
}

func (r *Runner) UpSQL(ctx context.Context, source FileSource) (RunReport, error) {
	loaded, err := loadSQLMigrations(source)
	if err != nil {
		return RunReport{}, err
	}
	pending, err := r.PendingSQL(ctx, source)
	if err != nil {
		return RunReport{}, err
	}
	loadedByKey := make(map[string]loadedSQLMigration, len(loaded))
	for _, migration := range loaded {
		loadedByKey[appliedRecordKey(migration.kind, migration.Version, migration.Description)] = migration
	}

	report := RunReport{Applied: make([]AppliedRecord, 0, len(pending))}
	for _, migration := range pending {
		loadedMigration, ok := loadedByKey[appliedRecordKey(kindForSQLMigration(migration), migration.Version, migration.Description)]
		if !ok {
			return report, fmt.Errorf("dbx/migrate: loaded sql migration not found for version %s", migration.Version)
		}
		applied, err := r.applySQLMigration(ctx, loadedMigration)
		if err != nil {
			return report, err
		}
		report.Applied = append(report.Applied, applied)
	}
	return report, nil
}

func (r *Runner) applyGoMigration(ctx context.Context, migration Migration) (AppliedRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return AppliedRecord{}, err
	}

	if err := migration.Up(ctx, tx); err != nil {
		_ = tx.Rollback()
		return AppliedRecord{}, err
	}

	record := AppliedRecord{
		Version:     migration.Version(),
		Description: migration.Description(),
		Kind:        KindGo,
		AppliedAt:   time.Now().UTC(),
		Checksum:    checksumGoMigration(migration),
		Success:     true,
	}
	if err := replaceAppliedRecord(ctx, tx, r.dialect, r.options.HistoryTable, record); err != nil {
		_ = tx.Rollback()
		return AppliedRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return AppliedRecord{}, err
	}
	return record, nil
}

func (r *Runner) applySQLMigration(ctx context.Context, migration loadedSQLMigration) (AppliedRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return AppliedRecord{}, err
	}

	if _, err := tx.ExecContext(ctx, migration.upSQL); err != nil {
		_ = tx.Rollback()
		return AppliedRecord{}, err
	}

	record := AppliedRecord{
		Version:     migration.Version,
		Description: migration.Description,
		Kind:        migration.kind,
		AppliedAt:   time.Now().UTC(),
		Checksum:    migration.checksum,
		Success:     true,
	}
	if err := replaceAppliedRecord(ctx, tx, r.dialect, r.options.HistoryTable, record); err != nil {
		_ = tx.Rollback()
		return AppliedRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return AppliedRecord{}, err
	}
	return record, nil
}

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
		return nil, err
	}
	loaded := make([]loadedSQLMigration, 0, len(items))
	for _, migration := range items {
		if migration.UpPath == "" {
			continue
		}
		upSQL, err := readSQLFile(source.FS, migration.UpPath)
		if err != nil {
			return nil, err
		}
		downSQL := ""
		if migration.DownPath != "" {
			downSQL, err = readSQLFile(source.FS, migration.DownPath)
			if err != nil {
				return nil, err
			}
		}
		loaded = append(loaded, loadedSQLMigration{
			SQLMigration: migration,
			kind:         kindForSQLMigration(migration),
			upSQL:        upSQL,
			downSQL:      downSQL,
			checksum:     checksumSQLMigration(migration, upSQL, downSQL),
		})
	}
	return loaded, nil
}

func readSQLFile(fsys fs.FS, path string) (string, error) {
	bytes, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}

func kindForSQLMigration(migration SQLMigration) Kind {
	if migration.Repeatable {
		return KindRepeatable
	}
	return KindSQL
}
