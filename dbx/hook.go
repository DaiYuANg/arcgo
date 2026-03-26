package dbx

import (
	"context"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
)

type Operation string

const (
	OperationQuery       Operation = "query"
	OperationExec        Operation = "exec"
	OperationQueryRow    Operation = "query_row"
	OperationBeginTx     Operation = "begin_tx"
	OperationCommitTx    Operation = "commit_tx"
	OperationRollbackTx  Operation = "rollback_tx"
	OperationAutoMigrate Operation = "auto_migrate"
	OperationValidate    Operation = "validate_schema"
)

// HookEvent carries operation details through Before/After hooks.
// Use it for logging, metrics, tracing, and slow-query detection.
type HookEvent struct {
	Operation       Operation
	Statement       string
	SQL             string
	Args            []any
	Table           string
	StartedAt       time.Time // Set in Before, use with Duration for slow-query detection.
	Duration        time.Duration
	RowsAffected    int64
	HasRowsAffected bool
	Err             error

	// Metadata holds arbitrary key-value pairs (e.g. trace_id, request_id) for observability.
	// Hooks can set it in Before and read it in After; values are included in logs when present.
	// Use SetMetadata to avoid nil map panics.
	Metadata map[string]any
}

// SetMetadata sets a key-value pair in Metadata, initializing the map if needed.
func (e *HookEvent) SetMetadata(key string, value any) {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
}

type Hook interface {
	Before(context.Context, *HookEvent) (context.Context, error)
	After(context.Context, *HookEvent)
}

type HookFuncs struct {
	BeforeFunc func(context.Context, *HookEvent) (context.Context, error)
	AfterFunc  func(context.Context, *HookEvent)
}

func (h HookFuncs) Before(ctx context.Context, event *HookEvent) (context.Context, error) {
	if h.BeforeFunc == nil {
		return ctx, nil
	}
	return h.BeforeFunc(ctx, event)
}

func (h HookFuncs) After(ctx context.Context, event *HookEvent) {
	if h.AfterFunc != nil {
		h.AfterFunc(ctx, event)
	}
}

type runtimeObserver struct {
	logger *slog.Logger
	hooks  []Hook
	debug  bool
}

func newRuntimeObserver(opts options) runtimeObserver {
	hooks := make([]Hook, len(opts.hooks))
	copy(hooks, opts.hooks)
	return runtimeObserver{
		logger: opts.logger,
		hooks:  hooks,
		debug:  opts.debug,
	}
}

func (o runtimeObserver) before(ctx context.Context, event HookEvent) (context.Context, *HookEvent, error) {
	copiedArgs := make([]any, len(event.Args))
	copy(copiedArgs, event.Args)
	event.Args = copiedArgs
	event.StartedAt = time.Now()

	for _, hook := range o.hooks {
		var err error
		ctx, err = hook.Before(ctx, &event)
		if err != nil {
			event.Err = err
			return ctx, &event, err
		}
	}
	return ctx, &event, nil
}

func (o runtimeObserver) after(ctx context.Context, event *HookEvent) {
	if event == nil {
		return
	}
	if event.StartedAt.IsZero() {
		event.StartedAt = time.Now()
	}
	if event.Duration == 0 {
		event.Duration = time.Since(event.StartedAt)
	}

	o.log(*event)
	for _, hook := range o.hooks {
		hook.After(ctx, event)
	}
}

func (o runtimeObserver) log(event HookEvent) {
	if o.logger == nil {
		return
	}
	if !o.debug && event.Err == nil {
		return
	}

	attrs := collectionx.NewListWithCapacity[any](14,
		"operation", event.Operation,
		"duration", event.Duration,
	)
	if event.Table != "" {
		attrs.Add("table", event.Table)
	}
	if event.Statement != "" {
		attrs.Add("statement", event.Statement)
	}
	if event.SQL != "" {
		attrs.Add("sql", event.SQL)
	}
	if len(event.Args) > 0 {
		attrs.Add("args", event.Args)
	}
	if event.HasRowsAffected {
		attrs.Add("rows_affected", event.RowsAffected)
	}
	for k, v := range event.Metadata {
		attrs.Add(k, v)
	}
	if event.Err != nil {
		attrs.Add("error", event.Err)
		o.logger.Error("dbx operation failed", attrs.Values()...)
		return
	}
	o.logger.Debug("dbx operation", attrs.Values()...)
}
