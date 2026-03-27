package migrate

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/pressly/goose/v3"
)

// UpGo applies the provided Go migrations.
func (r *Runner) UpGo(ctx context.Context, migrations ...Migration) (RunReport, error) {
	bundle, err := r.newRunnerEngineForGo(migrations)
	if err != nil {
		return RunReport{}, err
	}
	if bundle.engine == nil {
		return RunReport{}, nil
	}

	results, err := bundle.engine.Up(ctx)
	if err != nil {
		return RunReport{}, fmt.Errorf("dbx/migrate: apply go migrations: %w", err)
	}
	applied, err := r.Applied(ctx)
	if err != nil {
		return RunReport{}, err
	}
	return buildRunReport(applied, bundle.metaByVersion, results)
}

// UpSQL applies versioned and repeatable SQL migrations from source.
func (r *Runner) UpSQL(ctx context.Context, source FileSource) (RunReport, error) {
	bundle, repeatables, err := r.newRunnerEngineForSQL(source)
	if err != nil {
		return RunReport{}, err
	}

	report := RunReport{Applied: make([]AppliedRecord, 0, 8)}
	applied, err := r.versionedSQLRunReport(ctx, bundle)
	if err != nil {
		return report, err
	}
	report.Applied = append(report.Applied, applied...)

	indexed, err := r.appliedIndex(ctx)
	if err != nil {
		return report, err
	}
	repeatableRecords, err := r.applyPendingRepeatables(ctx, repeatables, indexed)
	if err != nil {
		return report, err
	}
	report.Applied = append(report.Applied, repeatableRecords...)
	return report, nil
}

func (r *Runner) versionedSQLRunReport(ctx context.Context, bundle *runnerEngine) ([]AppliedRecord, error) {
	if bundle == nil || bundle.engine == nil {
		return nil, nil
	}

	results, err := bundle.engine.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("dbx/migrate: apply sql migrations: %w", err)
	}
	applied, err := r.Applied(ctx)
	if err != nil {
		return nil, err
	}
	report, err := buildRunReport(applied, bundle.metaByVersion, results)
	if err != nil {
		return nil, err
	}
	return report.Applied, nil
}

func buildRunReport(
	applied []AppliedRecord,
	metaByVersion collectionx.Map[int64, AppliedRecord],
	results []*goose.MigrationResult,
) (RunReport, error) {
	report := RunReport{Applied: make([]AppliedRecord, 0, len(results))}
	for i := range results {
		result := results[i]
		record, ok := metaByVersion.Get(result.Source.Version)
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

func (r *Runner) applyPendingRepeatables(
	ctx context.Context,
	repeatables []loadedSQLMigration,
	indexed map[string]AppliedRecord,
) ([]AppliedRecord, error) {
	applied := collectionx.NewListWithCapacity[AppliedRecord](len(repeatables))
	for i := range repeatables {
		migration := repeatables[i]
		if !shouldApplyRepeatableMigration(migration, indexed) {
			continue
		}
		record, err := r.applySQLMigration(ctx, migration)
		if err != nil {
			return nil, err
		}
		applied.Add(record)
	}
	return applied.Values(), nil
}

func shouldApplyRepeatableMigration(migration loadedSQLMigration, indexed map[string]AppliedRecord) bool {
	key := appliedRecordKey(migration.kind, migration.Version, migration.Description)
	record, ok := indexed[key]
	return !ok || record.Checksum != migration.checksum
}

func (r *Runner) applySQLMigration(ctx context.Context, migration loadedSQLMigration) (_ AppliedRecord, resultErr error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return AppliedRecord{}, fmt.Errorf("dbx/migrate: begin sql migration %s transaction: %w", migration.Version, err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil && resultErr == nil {
			resultErr = fmt.Errorf("dbx/migrate: rollback sql migration %s: %w", migration.Version, rollbackErr)
		}
	}()

	if _, err := tx.ExecContext(ctx, migration.upSQL); err != nil {
		return AppliedRecord{}, fmt.Errorf("dbx/migrate: execute sql migration %s: %w", migration.Version, err)
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
		return AppliedRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return AppliedRecord{}, fmt.Errorf("dbx/migrate: commit sql migration %s: %w", migration.Version, err)
	}
	committed = true
	return record, nil
}
