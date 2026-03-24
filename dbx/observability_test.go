package dbx

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type memoryHandler struct {
	mu      sync.Mutex
	records []memoryRecord
}

type memoryRecord struct {
	level   slog.Level
	message string
	attrs   map[string]any
}

func (h *memoryHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *memoryHandler) Handle(_ context.Context, record slog.Record) error {
	entry := memoryRecord{
		level:   record.Level,
		message: record.Message,
		attrs:   make(map[string]any, record.NumAttrs()),
	}
	record.Attrs(func(attr slog.Attr) bool {
		entry.attrs[attr.Key] = attr.Value.Any()
		return true
	})

	h.mu.Lock()
	h.records = append(h.records, entry)
	h.mu.Unlock()
	return nil
}

func (h *memoryHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *memoryHandler) WithGroup(string) slog.Handler      { return h }

func findRecordByAttr(records []memoryRecord, key string, expected any) (memoryRecord, bool) {
	for _, record := range records {
		if value, ok := record.attrs[key]; ok && value == expected {
			return record, true
		}
	}
	return memoryRecord{}, false
}

func TestDBDebugLoggingAndHooks(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLiteWithSchema(t, `INSERT INTO "roles" ("id","name") VALUES (9,'admin')`)
	defer cleanup()

	handler := &memoryHandler{records: make([]memoryRecord, 0, 4)}
	logger := slog.New(handler)
	beforeCount := 0
	afterCount := 0

	db := NewWithOptions(
		sqlDB,
		testSQLiteDialect{},
		WithLogger(logger),
		WithDebug(true),
		WithHooks(HookFuncs{
			BeforeFunc: func(ctx context.Context, event *HookEvent) (context.Context, error) {
				beforeCount++
				if event.Operation != OperationExec {
					t.Fatalf("unexpected hook operation: %s", event.Operation)
				}
				return ctx, nil
			},
			AfterFunc: func(_ context.Context, event *HookEvent) {
				afterCount++
				if event.Err != nil {
					t.Fatalf("unexpected hook error: %v", event.Err)
				}
			},
		}),
	)

	users := MustSchema("users", UserSchema{})
	mapper := MustMapper[User](users)
	entity := &User{Username: "alice", Email: "alice@example.com", Status: 1, RoleID: 9}
	assignments, err := mapper.InsertAssignments(users, entity)
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}

	if _, err := Exec(context.Background(), db, InsertInto(users).Values(assignments...)); err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}

	if beforeCount != 1 || afterCount != 1 {
		t.Fatalf("unexpected hook counts: before=%d after=%d", beforeCount, afterCount)
	}
	if len(handler.records) == 0 {
		t.Fatal("expected debug log record")
	}
	record, ok := findRecordByAttr(handler.records, "operation", OperationExec)
	if !ok {
		t.Fatalf("expected operation log record, got %#v", handler.records)
	}
	if record.level != slog.LevelDebug {
		t.Fatalf("unexpected log level: %v", record.level)
	}
	if record.attrs["operation"] != OperationExec {
		t.Fatalf("unexpected log attrs: %#v", record.attrs)
	}
	if record.attrs["sql"] == "" {
		t.Fatalf("expected sql log attr, got %#v", record.attrs)
	}
}

func TestSchemaOperationsEmitObserverEvents(t *testing.T) {
	handler := &memoryHandler{records: make([]memoryRecord, 0, 8)}
	logger := slog.New(handler)
	beforeOps := make([]Operation, 0, 2)
	afterOps := make([]Operation, 0, 2)

	users := MustSchema("users", UserSchema{})
	schemaDialect := newFakeSchemaDialect()
	spec := buildTableSpec(users.schemaRef())
	schemaDialect.tables["users"] = TableState{
		Exists:      true,
		Name:        "users",
		Columns:     []ColumnState{toColumnState(users.Columns()[0]), toColumnState(users.Columns()[1]), toColumnState(users.Columns()[2]), toColumnState(users.Columns()[3]), toColumnState(users.Columns()[4])},
		Indexes:     toIndexStates(spec.Indexes),
		PrimaryKey:  &PrimaryKeyState{Name: spec.PrimaryKey.Name, Columns: append([]string(nil), spec.PrimaryKey.Columns...)},
		ForeignKeys: toForeignKeyStates(spec.ForeignKeys),
	}

	db := NewWithOptions(
		nil,
		schemaDialect,
		WithLogger(logger),
		WithDebug(true),
		WithHooks(HookFuncs{
			BeforeFunc: func(ctx context.Context, event *HookEvent) (context.Context, error) {
				beforeOps = append(beforeOps, event.Operation)
				return ctx, nil
			},
			AfterFunc: func(_ context.Context, event *HookEvent) {
				afterOps = append(afterOps, event.Operation)
			},
		}),
	)

	if _, err := db.ValidateSchemas(context.Background(), users); err != nil {
		t.Fatalf("ValidateSchemas returned error: %v", err)
	}
	if _, err := db.AutoMigrate(context.Background(), users); err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}

	if len(beforeOps) != 2 || beforeOps[0] != OperationValidate || beforeOps[1] != OperationAutoMigrate {
		t.Fatalf("unexpected before ops: %#v", beforeOps)
	}
	if len(afterOps) != 2 || afterOps[0] != OperationValidate || afterOps[1] != OperationAutoMigrate {
		t.Fatalf("unexpected after ops: %#v", afterOps)
	}
	if len(handler.records) < 2 {
		t.Fatalf("expected schema operation logs, got %d", len(handler.records))
	}
}

func TestHookEventMetadataAndDuration(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLiteWithSchema(t, `INSERT INTO "roles" ("id","name") VALUES (1,'user')`)
	defer cleanup()

	handler := &memoryHandler{records: make([]memoryRecord, 0, 4)}
	logger := slog.New(handler)
	var afterEvent *HookEvent

	users := MustSchema("users", UserSchema{})
	mapper := MustMapper[User](users)
	entity := &User{Username: "bob", Email: "bob@example.com", Status: 1, RoleID: 1}
	assignments, err := mapper.InsertAssignments(users, entity)
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}

	db := NewWithOptions(
		sqlDB,
		testSQLiteDialect{},
		WithLogger(logger),
		WithDebug(true),
		WithHooks(HookFuncs{
			BeforeFunc: func(ctx context.Context, event *HookEvent) (context.Context, error) {
				event.SetMetadata("trace_id", "abc-123")
				event.SetMetadata("request_id", "req-456")
				return ctx, nil
			},
			AfterFunc: func(_ context.Context, event *HookEvent) {
				afterEvent = event
			},
		}),
	)

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "abc-123")
	if _, err := Exec(ctx, db, InsertInto(users).Values(assignments...)); err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}

	if afterEvent == nil {
		t.Fatal("AfterFunc was not called")
	}
	if afterEvent.Metadata["trace_id"] != "abc-123" {
		t.Fatalf("unexpected trace_id: %v", afterEvent.Metadata["trace_id"])
	}
	if afterEvent.Metadata["request_id"] != "req-456" {
		t.Fatalf("unexpected request_id: %v", afterEvent.Metadata["request_id"])
	}
	if afterEvent.StartedAt.IsZero() {
		t.Fatal("expected StartedAt to be set")
	}
	// StartedAt can equal time.Now() within the same wall-clock tick; require not in the future.
	if afterEvent.StartedAt.After(time.Now()) {
		t.Fatalf("expected StartedAt <= now: %v", afterEvent.StartedAt)
	}
	if afterEvent.Duration < 0 {
		t.Fatalf("expected non-negative Duration: %v", afterEvent.Duration)
	}

	if len(handler.records) == 0 {
		t.Fatal("expected debug log record")
	}
	record, ok := findRecordByAttr(handler.records, "operation", OperationExec)
	if !ok {
		t.Fatalf("expected operation log record, got %#v", handler.records)
	}
	if record.attrs["trace_id"] != "abc-123" {
		t.Fatalf("expected trace_id in log attrs: %#v", record.attrs)
	}
	if record.attrs["request_id"] != "req-456" {
		t.Fatalf("expected request_id in log attrs: %#v", record.attrs)
	}
}
