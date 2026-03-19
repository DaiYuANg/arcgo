package dbx

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/dbx/internal/testsql"
)

type codecPreferences struct {
	Theme string   `json:"theme"`
	Flags []string `json:"flags"`
}

type codecRecord struct {
	ID          int64            `dbx:"id"`
	Preferences codecPreferences `dbx:"preferences,codec=json"`
	Tags        []string         `dbx:"tags,codec=csv"`
}

type codecSchema struct {
	Schema[codecRecord]
	ID          Column[codecRecord, int64]            `dbx:"id,pk,auto"`
	Preferences Column[codecRecord, codecPreferences] `dbx:"preferences"`
	Tags        Column[codecRecord, []string]         `dbx:"tags"`
}

type scopedCodecRecord struct {
	ID   int64    `dbx:"id"`
	Tags []string `dbx:"tags,codec=scoped_csv"`
}

type scopedCodecSchema struct {
	Schema[scopedCodecRecord]
	ID   Column[scopedCodecRecord, int64]    `dbx:"id,pk,auto"`
	Tags Column[scopedCodecRecord, []string] `dbx:"tags"`
}

type timeCodecRecord struct {
	ID        int64     `dbx:"id"`
	CreatedAt time.Time `dbx:"created_at,codec=unix_milli_time"`
}

type timeCodecSchema struct {
	Schema[timeCodecRecord]
	ID        Column[timeCodecRecord, int64]     `dbx:"id,pk,auto"`
	CreatedAt Column[timeCodecRecord, time.Time] `dbx:"created_at"`
}

type accountStatus string

const (
	accountStatusActive  accountStatus = "active"
	accountStatusBlocked accountStatus = "blocked"
)

func (s accountStatus) MarshalText() ([]byte, error) {
	switch s {
	case accountStatusActive, accountStatusBlocked:
		return []byte(string(s)), nil
	default:
		return nil, errors.New("dbx: invalid account status")
	}
}

func (s *accountStatus) UnmarshalText(text []byte) error {
	value := accountStatus(strings.ToLower(strings.TrimSpace(string(text))))
	switch value {
	case accountStatusActive, accountStatusBlocked:
		*s = value
		return nil
	default:
		return errors.New("dbx: invalid account status")
	}
}

type decimalAmount struct {
	text string
}

func (a decimalAmount) MarshalText() ([]byte, error) {
	if strings.TrimSpace(a.text) == "" {
		return nil, errors.New("dbx: empty decimal amount")
	}
	return []byte(a.text), nil
}

func (a *decimalAmount) UnmarshalText(text []byte) error {
	trimmed := strings.TrimSpace(string(text))
	if trimmed == "" {
		return errors.New("dbx: empty decimal amount")
	}
	a.text = trimmed
	return nil
}

func (a decimalAmount) String() string {
	return a.text
}

type textCodecRecord struct {
	ID      int64         `dbx:"id"`
	Status  accountStatus `dbx:"status,codec=text"`
	Balance decimalAmount `dbx:"balance,codec=text"`
}

type textCodecSchema struct {
	Schema[textCodecRecord]
	ID      Column[textCodecRecord, int64]         `dbx:"id,pk,auto"`
	Status  Column[textCodecRecord, accountStatus] `dbx:"status,type=text"`
	Balance Column[textCodecRecord, decimalAmount] `dbx:"balance,type=text"`
}

var registerCSVCodecOnce sync.Once

func registerCSVCodec(t *testing.T) {
	t.Helper()
	registerCSVCodecOnce.Do(func() {
		MustRegisterCodec(NewCodec[[]string](
			"csv",
			func(src any) ([]string, error) {
				switch value := src.(type) {
				case string:
					return splitCSV(value), nil
				case []byte:
					return splitCSV(string(value)), nil
				default:
					return nil, errors.New("dbx: csv codec only supports string or []byte")
				}
			},
			func(values []string) (any, error) {
				return strings.Join(values, ","), nil
			},
		))
	})
}

func TestStructMapperScansCodecFields(t *testing.T) {
	registerCSVCodec(t)

	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "codec_accounts"."id", "codec_accounts"."preferences", "codec_accounts"."tags" FROM "codec_accounts"`,
				Columns: []string{"id", "preferences", "tags"},
				Rows: [][]driver.Value{
					{int64(1), `{"theme":"dark","flags":["alpha","beta"]}`, "go,dbx,orm"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	accounts := MustSchema("codec_accounts", codecSchema{})
	items, err := QueryAll(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		Select(accounts.AllColumns()...).From(accounts),
		MustStructMapper[codecRecord](),
	)
	if err != nil {
		t.Fatalf("QueryAll returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].Preferences.Theme != "dark" {
		t.Fatalf("unexpected preferences: %+v", items[0].Preferences)
	}
	if len(items[0].Tags) != 3 || items[0].Tags[1] != "dbx" {
		t.Fatalf("unexpected tags: %+v", items[0].Tags)
	}
}

func TestMapperAssignmentsUseCodecEncoding(t *testing.T) {
	registerCSVCodec(t)

	expectedJSON := []byte(`{"theme":"dark","flags":["admin","beta"]}`)
	sqlDB, recorder, cleanup, err := testsql.Open(testsql.Plan{
		Execs: []testsql.ExecPlan{
			{
				SQL:          `INSERT INTO "codec_accounts" ("preferences", "tags") VALUES (?, ?)`,
				Args:         []driver.Value{expectedJSON, "alpha,beta"},
				RowsAffected: 1,
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	accounts := MustSchema("codec_accounts", codecSchema{})
	mapper := MustMapper[codecRecord](accounts)
	entity := &codecRecord{
		Preferences: codecPreferences{
			Theme: "dark",
			Flags: []string{"admin", "beta"},
		},
		Tags: []string{"alpha", "beta"},
	}

	assignments, err := mapper.InsertAssignments(accounts, entity)
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("unexpected assignment count: %d", len(assignments))
	}

	if _, err := Exec(context.Background(), New(sqlDB, testSQLiteDialect{}), InsertInto(accounts).Values(assignments...)); err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if len(recorder.Execs) != 1 {
		t.Fatalf("unexpected exec count: %d", len(recorder.Execs))
	}
}

func TestNewStructMapperReturnsErrorForUnknownCodec(t *testing.T) {
	type invalidCodecRecord struct {
		ID   int64  `dbx:"id"`
		Data string `dbx:"data,codec=missing"`
	}

	_, err := NewStructMapper[invalidCodecRecord]()
	if !errors.Is(err, ErrUnknownCodec) {
		t.Fatalf("expected ErrUnknownCodec, got: %v", err)
	}
}

func TestStructMapperWithOptionsUsesScopedCodecRegistry(t *testing.T) {
	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "scoped_codec_records"."id", "scoped_codec_records"."tags" FROM "scoped_codec_records"`,
				Columns: []string{"id", "tags"},
				Rows: [][]driver.Value{
					{int64(1), "one,two"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	schema := MustSchema("scoped_codec_records", scopedCodecSchema{})
	scopedCSV := NewCodec[[]string](
		"scoped_csv",
		func(src any) ([]string, error) {
			switch value := src.(type) {
			case string:
				return splitCSV(value), nil
			case []byte:
				return splitCSV(string(value)), nil
			default:
				return nil, errors.New("dbx: scoped csv codec only supports string or []byte")
			}
		},
		func(values []string) (any, error) {
			return strings.Join(values, ","), nil
		},
	)

	mapper, err := NewStructMapperWithOptions[scopedCodecRecord](WithMapperCodecs(scopedCSV))
	if err != nil {
		t.Fatalf("NewStructMapperWithOptions returned error: %v", err)
	}

	items, err := QueryAll(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		Select(schema.AllColumns()...).From(schema),
		mapper,
	)
	if err != nil {
		t.Fatalf("QueryAll returned error: %v", err)
	}
	if len(items) != 1 || len(items[0].Tags) != 2 || items[0].Tags[1] != "two" {
		t.Fatalf("unexpected scoped codec items: %+v", items)
	}

	if _, err := NewStructMapper[scopedCodecRecord](); !errors.Is(err, ErrUnknownCodec) {
		t.Fatalf("expected default mapper to reject scoped codec tag, got: %v", err)
	}
}

func TestBuiltInUnixMilliTimeCodecScanAndEncode(t *testing.T) {
	createdAt := time.UnixMilli(1711111111222).UTC()

	sqlDB, recorder, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "time_codec_records"."id", "time_codec_records"."created_at" FROM "time_codec_records"`,
				Columns: []string{"id", "created_at"},
				Rows: [][]driver.Value{
					{int64(1), createdAt.UnixMilli()},
				},
			},
		},
		Execs: []testsql.ExecPlan{
			{
				SQL:          `INSERT INTO "time_codec_records" ("created_at") VALUES (?)`,
				Args:         []driver.Value{createdAt.UnixMilli()},
				RowsAffected: 1,
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	schema := MustSchema("time_codec_records", timeCodecSchema{})
	mapper := MustMapper[timeCodecRecord](schema)

	items, err := QueryAll(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		Select(schema.AllColumns()...).From(schema),
		mapper,
	)
	if err != nil {
		t.Fatalf("QueryAll returned error: %v", err)
	}
	if len(items) != 1 || !items[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected time codec items: %+v", items)
	}

	assignments, err := mapper.InsertAssignments(schema, &items[0])
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}
	if _, err := Exec(context.Background(), New(sqlDB, testSQLiteDialect{}), InsertInto(schema).Values(assignments...)); err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if len(recorder.Execs) != 1 {
		t.Fatalf("unexpected exec count: %d", len(recorder.Execs))
	}
}

func TestBuiltInTextCodecScanAndEncode(t *testing.T) {
	sqlDB, recorder, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "text_codec_records"."id", "text_codec_records"."status", "text_codec_records"."balance" FROM "text_codec_records"`,
				Columns: []string{"id", "status", "balance"},
				Rows: [][]driver.Value{
					{int64(1), "active", "123.45"},
				},
			},
		},
		Execs: []testsql.ExecPlan{
			{
				SQL:          `INSERT INTO "text_codec_records" ("status", "balance") VALUES (?, ?)`,
				Args:         []driver.Value{"active", "123.45"},
				RowsAffected: 1,
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	schema := MustSchema("text_codec_records", textCodecSchema{})
	mapper := MustMapper[textCodecRecord](schema)

	items, err := QueryAll(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		Select(schema.AllColumns()...).From(schema),
		mapper,
	)
	if err != nil {
		t.Fatalf("QueryAll returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].Status != accountStatusActive {
		t.Fatalf("unexpected status: %q", items[0].Status)
	}
	if items[0].Balance.String() != "123.45" {
		t.Fatalf("unexpected balance: %s", items[0].Balance.String())
	}

	assignments, err := mapper.InsertAssignments(schema, &items[0])
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}
	if _, err := Exec(context.Background(), New(sqlDB, testSQLiteDialect{}), InsertInto(schema).Values(assignments...)); err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if len(recorder.Execs) != 1 {
		t.Fatalf("unexpected exec count: %d", len(recorder.Execs))
	}
}

func splitCSV(input string) []string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}
