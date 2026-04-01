package migrate

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/pressly/goose/v3"
	"github.com/samber/lo"
)

// PendingGo returns Go migrations that have not yet been applied.
func (r *Runner) PendingGo(ctx context.Context, migrations ...Migration) ([]Migration, error) {
	bundle, err := r.newRunnerEngineForGo(migrations)
	if err != nil {
		return nil, err
	}
	if bundle.engine == nil {
		return nil, nil
	}

	statuses, err := pendingStatuses(ctx, bundle.engine, "go")
	if err != nil {
		return nil, err
	}
	indexed, err := r.appliedIndex(ctx)
	if err != nil {
		return nil, err
	}
	byVersion, err := indexGoMigrationsByVersion(migrations)
	if err != nil {
		return nil, err
	}

	return collectPendingGoMigrations(statuses, bundle.metaByVersion, indexed, byVersion, r.options.ValidateHash)
}

// PendingSQL returns SQL migrations that should be applied next.
func (r *Runner) PendingSQL(ctx context.Context, source FileSource) ([]SQLMigration, error) {
	bundle, repeatables, err := r.newRunnerEngineForSQL(source)
	if err != nil {
		return nil, err
	}
	indexed, err := r.appliedIndex(ctx)
	if err != nil {
		return nil, err
	}

	pending := collectionx.NewList[SQLMigration]()
	if bundle != nil && bundle.engine != nil {
		versionedPending, pendingErr := r.pendingVersionedSQL(ctx, source, bundle, indexed)
		if pendingErr != nil {
			return nil, pendingErr
		}
		pending.Add(versionedPending...)
	}

	pending.Add(pendingRepeatableMigrations(repeatables, indexed)...)
	return pending.Values(), nil
}

func pendingStatuses(ctx context.Context, engine *goose.Provider, kind string) ([]*goose.MigrationStatus, error) {
	if _, err := engine.HasPending(ctx); err != nil {
		return nil, fmt.Errorf("dbx/migrate: check %s migration pending state: %w", kind, err)
	}

	statuses, err := engine.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("dbx/migrate: load %s migration status: %w", kind, err)
	}
	return statuses, nil
}

func (r *Runner) appliedIndex(ctx context.Context) (map[string]AppliedRecord, error) {
	applied, err := r.Applied(ctx)
	if err != nil {
		return nil, err
	}
	return indexAppliedRecords(applied), nil
}

func indexGoMigrationsByVersion(migrations []Migration) (map[int64]Migration, error) {
	byVersion, err := lo.ReduceErr(migrations, func(result map[int64]Migration, migration Migration, _ int) (map[int64]Migration, error) {
		version, parseErr := parseNumericVersion(migration.Version())
		if parseErr != nil {
			return nil, fmt.Errorf("dbx/migrate: parse go migration version %q: %w", migration.Version(), parseErr)
		}
		result[version] = migration
		return result, nil
	}, make(map[int64]Migration, len(migrations)))
	if err != nil {
		return nil, fmt.Errorf("dbx/migrate: index sql migrations by version: %w", err)
	}
	return byVersion, nil
}

func collectPendingGoMigrations(
	statuses []*goose.MigrationStatus,
	metaByVersion collectionx.Map[int64, AppliedRecord],
	indexed map[string]AppliedRecord,
	byVersion map[int64]Migration,
	validateHash bool,
) ([]Migration, error) {
	pending, err := lo.ReduceErr(statuses, func(result []Migration, status *goose.MigrationStatus, _ int) ([]Migration, error) {
		migration, ok := byVersion[status.Source.Version]
		if !ok {
			return result, nil
		}
		if err := validatePendingStatus(status, metaByVersion, indexed, validateHash); err != nil {
			return nil, err
		}
		if status.State != goose.StatePending {
			return result, nil
		}
		return lo.Concat(result, []Migration{migration}), nil
	}, make([]Migration, 0, len(statuses)))
	if err != nil {
		return nil, fmt.Errorf("dbx/migrate: collect pending go migrations: %w", err)
	}
	return pending, nil
}

func (r *Runner) pendingVersionedSQL(
	ctx context.Context,
	source FileSource,
	bundle *runnerEngine,
	indexed map[string]AppliedRecord,
) ([]SQLMigration, error) {
	statuses, err := pendingStatuses(ctx, bundle.engine, "sql")
	if err != nil {
		return nil, err
	}

	versionedByVersion, err := indexVersionedSQLMigrations(source)
	if err != nil {
		return nil, err
	}
	return collectPendingSQLMigrations(statuses, bundle.metaByVersion, indexed, versionedByVersion, r.options.ValidateHash)
}

func indexVersionedSQLMigrations(source FileSource) (map[int64]SQLMigration, error) {
	loaded, err := loadSQLMigrations(source)
	if err != nil {
		return nil, err
	}

	byVersion, err := lo.ReduceErr(loaded, func(result map[int64]SQLMigration, migration loadedSQLMigration, _ int) (map[int64]SQLMigration, error) {
		if migration.Repeatable {
			return result, nil
		}
		version, parseErr := parseNumericVersion(migration.Version)
		if parseErr != nil {
			return nil, fmt.Errorf("dbx/migrate: parse sql migration version %q: %w", migration.Version, parseErr)
		}
		result[version] = migration.SQLMigration
		return result, nil
	}, make(map[int64]SQLMigration, len(loaded)))
	if err != nil {
		return nil, fmt.Errorf("dbx/migrate: index sql migrations by version: %w", err)
	}
	return byVersion, nil
}

func collectPendingSQLMigrations(
	statuses []*goose.MigrationStatus,
	metaByVersion collectionx.Map[int64, AppliedRecord],
	indexed map[string]AppliedRecord,
	byVersion map[int64]SQLMigration,
	validateHash bool,
) ([]SQLMigration, error) {
	pending, err := lo.ReduceErr(statuses, func(result []SQLMigration, status *goose.MigrationStatus, _ int) ([]SQLMigration, error) {
		migration, ok := byVersion[status.Source.Version]
		if !ok {
			return result, nil
		}
		if err := validatePendingStatus(status, metaByVersion, indexed, validateHash); err != nil {
			return nil, err
		}
		if status.State != goose.StatePending {
			return result, nil
		}
		return lo.Concat(result, []SQLMigration{migration}), nil
	}, make([]SQLMigration, 0, len(statuses)))
	if err != nil {
		return nil, fmt.Errorf("dbx/migrate: collect pending sql migrations: %w", err)
	}
	return pending, nil
}

func validatePendingStatus(
	status *goose.MigrationStatus,
	metaByVersion collectionx.Map[int64, AppliedRecord],
	indexed map[string]AppliedRecord,
	validateHash bool,
) error {
	if !validateHash || status.State == goose.StatePending {
		return nil
	}

	record, ok := metaByVersion.Get(status.Source.Version)
	if !ok {
		return nil
	}
	existing, exists := indexed[appliedRecordKey(record.Kind, record.Version, record.Description)]
	if exists && existing.Checksum != record.Checksum {
		return fmt.Errorf("dbx/migrate: migration checksum mismatch for version %s", record.Version)
	}
	return nil
}

func pendingRepeatableMigrations(repeatables []loadedSQLMigration, indexed map[string]AppliedRecord) []SQLMigration {
	return lo.FilterMap(repeatables, func(migration loadedSQLMigration, _ int) (SQLMigration, bool) {
		key := appliedRecordKey(migration.kind, migration.Version, migration.Description)
		record, ok := indexed[key]
		if ok && record.Checksum == migration.checksum {
			return SQLMigration{}, false
		}
		return migration.SQLMigration, true
	})
}
