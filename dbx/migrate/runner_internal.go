package migrate

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"

	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

const timeLayout = "2006-01-02T15:04:05.999999999Z07:00"

func appliedRecordKey(kind Kind, version, description string) string {
	return string(kind) + "\x1f" + version + "\x1f" + description
}

func indexAppliedRecords(records []AppliedRecord) map[string]AppliedRecord {
	indexed := make(map[string]AppliedRecord, len(records))
	for _, record := range records {
		indexed[appliedRecordKey(record.Kind, record.Version, record.Description)] = record
	}
	return indexed
}

func hasAppliedNewerVersion(records []AppliedRecord, version string) bool {
	for _, record := range records {
		if record.Kind == KindRepeatable || record.Version == "" {
			continue
		}
		if record.Version > version {
			return true
		}
	}
	return false
}

func checksumGoMigration(migration Migration) string {
	return checksumString("go|" + migration.Version() + "|" + migration.Description())
}

func checksumSQLMigration(migration SQLMigration, upSQL, downSQL string) string {
	return checksumString(strings.Join([]string{
		string(kindForSQLMigration(migration)),
		migration.Version,
		migration.Description,
		upSQL,
		downSQL,
	}, "\n--dbx-migrate--\n"))
}

func checksumString(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

func historyTableDDL(d dialect.Dialect, table string) string {
	q := d.QuoteIdent
	return "CREATE TABLE IF NOT EXISTS " + q(table) + " (" +
		q("version") + " VARCHAR(255) NOT NULL, " +
		q("description") + " VARCHAR(255) NOT NULL, " +
		q("kind") + " VARCHAR(32) NOT NULL, " +
		q("checksum") + " VARCHAR(128) NOT NULL, " +
		q("success") + " BOOLEAN NOT NULL, " +
		q("applied_at") + " VARCHAR(64) NOT NULL, " +
		"PRIMARY KEY (" + q("version") + ", " + q("kind") + ", " + q("description") + "))"
}

func appliedRecordsSQL(d dialect.Dialect, table string) string {
	q := d.QuoteIdent
	return "SELECT " + q("version") + ", " + q("description") + ", " + q("kind") + ", " + q("applied_at") + ", " + q("checksum") + ", " + q("success") +
		" FROM " + q(table) +
		" ORDER BY " + q("applied_at") + ", " + q("version") + ", " + q("description")
}

func replaceAppliedRecord(ctx context.Context, tx *sql.Tx, d dialect.Dialect, table string, record AppliedRecord) error {
	q := d.QuoteIdent
	deleteSQL := "DELETE FROM " + q(table) +
		" WHERE " + q("version") + " = " + d.BindVar(1) +
		" AND " + q("kind") + " = " + d.BindVar(2) +
		" AND " + q("description") + " = " + d.BindVar(3)
	if _, err := tx.ExecContext(ctx, deleteSQL, record.Version, string(record.Kind), record.Description); err != nil {
		return err
	}

	insertSQL := "INSERT INTO " + q(table) +
		" (" + q("version") + ", " + q("description") + ", " + q("kind") + ", " + q("checksum") + ", " + q("success") + ", " + q("applied_at") + ")" +
		" VALUES (" + d.BindVar(1) + ", " + d.BindVar(2) + ", " + d.BindVar(3) + ", " + d.BindVar(4) + ", " + d.BindVar(5) + ", " + d.BindVar(6) + ")"
	_, err := tx.ExecContext(ctx, insertSQL,
		record.Version,
		record.Description,
		string(record.Kind),
		record.Checksum,
		record.Success,
		record.AppliedAt.UTC().Format(timeLayout),
	)
	return err
}
