package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/pressly/goose/v3"
	"github.com/samber/lo"
)

type RunReport struct {
	Applied []AppliedRecord
}

func (r *Runner) EnsureHistory(ctx context.Context) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}
	return newHistoryStore(r.dialect, r.options.HistoryTable, collectionx.NewMap[int64, AppliedRecord]()).CreateVersionTable(ctx, r.db)
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

	items := collectionx.NewListWithCapacity[AppliedRecord](8)
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
		items.Add(record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items.Values(), nil
}

func (r *Runner) PendingGo(ctx context.Context, migrations ...Migration) ([]Migration, error) {
	bundle, err := r.newRunnerEngineForGo(migrations)
	if err != nil {
		return nil, err
	}
	if bundle == nil || bundle.engine == nil {
		return nil, nil
	}
	if _, err := bundle.engine.HasPending(ctx); err != nil {
		return nil, err
	}

	statuses, err := bundle.engine.Status(ctx)
	if err != nil {
		return nil, err
	}
	applied, err := r.Applied(ctx)
	if err != nil {
		return nil, err
	}
	indexed := indexAppliedRecords(applied)
	byVersion := collectionx.NewMapWithCapacity[int64, Migration](len(migrations))
	for _, migration := range migrations {
		version, err := parseNumericVersion(migration.Version())
		if err != nil {
			return nil, err
		}
		byVersion.Set(version, migration)
	}

	pending := collectionx.NewListWithCapacity[Migration](len(migrations))
	for _, status := range statuses {
		migration, ok := byVersion.Get(status.Source.Version)
		if !ok {
			continue
		}
		record, ok := bundle.metaByVersion.Get(status.Source.Version)
		if ok && r.options.ValidateHash && status.State != goose.StatePending {
			existing, exists := indexed[appliedRecordKey(record.Kind, record.Version, record.Description)]
			if exists && existing.Checksum != record.Checksum {
				return nil, fmt.Errorf("dbx/migrate: go migration checksum mismatch for version %s", record.Version)
			}
		}
		if status.State == goose.StatePending {
			pending.Add(migration)
		}
	}
	return pending.Values(), nil
}

func (r *Runner) PendingSQL(ctx context.Context, source FileSource) ([]SQLMigration, error) {
	bundle, repeatables, err := r.newRunnerEngineForSQL(source)
	if err != nil {
		return nil, err
	}
	applied, err := r.Applied(ctx)
	if err != nil {
		return nil, err
	}
	indexed := indexAppliedRecords(applied)
	pending := collectionx.NewList[SQLMigration]()

	if bundle != nil && bundle.engine != nil {
		if _, err := bundle.engine.HasPending(ctx); err != nil {
			return nil, err
		}
		statuses, err := bundle.engine.Status(ctx)
		if err != nil {
			return nil, err
		}

		versionedByVersion := collectionx.NewMapWithCapacity[int64, SQLMigration](bundle.metaByVersion.Len())
		loaded, err := loadSQLMigrations(source)
		if err != nil {
			return nil, err
		}
		for _, migration := range loaded {
			if migration.Repeatable {
				continue
			}
			version, err := parseNumericVersion(migration.Version)
			if err != nil {
				return nil, err
			}
			versionedByVersion.Set(version, migration.SQLMigration)
		}

		for _, status := range statuses {
			migration, ok := versionedByVersion.Get(status.Source.Version)
			if !ok {
				continue
			}
			record, ok := bundle.metaByVersion.Get(status.Source.Version)
			if ok && r.options.ValidateHash && status.State != goose.StatePending {
				existing, exists := indexed[appliedRecordKey(record.Kind, record.Version, record.Description)]
				if exists && existing.Checksum != record.Checksum {
					return nil, fmt.Errorf("dbx/migrate: sql migration checksum mismatch for version %s", record.Version)
				}
			}
			if status.State == goose.StatePending {
				pending.Add(migration)
			}
		}
	}

	for _, migration := range repeatables {
		key := appliedRecordKey(migration.kind, migration.Version, migration.Description)
		record, ok := indexed[key]
		if ok {
			if record.Checksum != migration.checksum {
				pending.Add(migration.SQLMigration)
			}
			continue
		}
		pending.Add(migration.SQLMigration)
	}
	return pending.Values(), nil
}

func (r *Runner) UpGo(ctx context.Context, migrations ...Migration) (RunReport, error) {
	bundle, err := r.newRunnerEngineForGo(migrations)
	if err != nil {
		return RunReport{}, err
	}
	if bundle == nil || bundle.engine == nil {
		return RunReport{}, nil
	}

	results, err := bundle.engine.Up(ctx)
	if err != nil {
		return RunReport{}, err
	}
	applied, err := r.Applied(ctx)
	if err != nil {
		return RunReport{}, err
	}
	report := RunReport{Applied: make([]AppliedRecord, 0, len(results))}
	for _, result := range results {
		record, ok := bundle.metaByVersion.Get(result.Source.Version)
		if !ok {
			continue
		}
		current, err := appliedRecordForVersion(applied, record)
		if err != nil {
			return report, err
		}
		report.Applied = append(report.Applied, current)
	}
	return report, nil
}

func (r *Runner) UpSQL(ctx context.Context, source FileSource) (RunReport, error) {
	bundle, repeatables, err := r.newRunnerEngineForSQL(source)
	if err != nil {
		return RunReport{}, err
	}
	report := RunReport{Applied: make([]AppliedRecord, 0, 8)}
	var applied []AppliedRecord

	if bundle != nil && bundle.engine != nil {
		results, err := bundle.engine.Up(ctx)
		if err != nil {
			return report, err
		}
		applied, err = r.Applied(ctx)
		if err != nil {
			return report, err
		}
		for _, result := range results {
			record, ok := bundle.metaByVersion.Get(result.Source.Version)
			if !ok {
				continue
			}
			current, err := appliedRecordForVersion(applied, record)
			if err != nil {
				return report, err
			}
			report.Applied = append(report.Applied, current)
		}
	}

	if applied == nil {
		applied, err = r.Applied(ctx)
		if err != nil {
			return report, err
		}
	}
	indexed := indexAppliedRecords(applied)
	for _, migration := range repeatables {
		key := appliedRecordKey(migration.kind, migration.Version, migration.Description)
		record, ok := indexed[key]
		if ok && record.Checksum == migration.checksum {
			continue
		}
		appliedRecord, err := r.applySQLMigration(ctx, migration)
		if err != nil {
			return report, err
		}
		report.Applied = append(report.Applied, appliedRecord)
	}
	return report, nil
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
	for _, migration := range lo.Filter(items, func(m SQLMigration, _ int) bool { return m.UpPath != "" }) {
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
