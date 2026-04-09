package dix

import (
	"context"
	"fmt"
	"log/slog"

	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
)

// Event is an internal dix framework event emitted to EventLogger implementations.
type Event interface {
	dixEvent()
}

// EventLevel is the severity level for MessageEvent.
type EventLevel string

const (
	// EventLevelDebug is the debug severity.
	EventLevelDebug EventLevel = "debug"
	// EventLevelInfo is the info severity.
	EventLevelInfo EventLevel = "info"
	// EventLevelWarn is the warn severity.
	EventLevelWarn EventLevel = "warn"
	// EventLevelError is the error severity.
	EventLevelError EventLevel = "error"
)

// EventField is a structured field attached to MessageEvent.
type EventField struct {
	Key   string
	Value any
}

// MessageEvent carries structured dix framework log messages that do not map to a higher-level lifecycle event.
type MessageEvent struct {
	Level   EventLevel
	Message string
	Fields  *collectionlist.List[EventField]
}

// EventLogger receives all internal dix logging events.
type EventLogger interface {
	LogEvent(context.Context, Event)
}

type eventLoggerEnabler interface {
	Enabled(context.Context, EventLevel) bool
}

func (BuildEvent) dixEvent()           {}
func (StartEvent) dixEvent()           {}
func (StopEvent) dixEvent()            {}
func (HealthCheckEvent) dixEvent()     {}
func (StateTransitionEvent) dixEvent() {}
func (MessageEvent) dixEvent()         {}

// NewSlogEventLogger adapts a slog logger to the dix EventLogger interface.
func NewSlogEventLogger(logger *slog.Logger) EventLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &slogEventLogger{logger: logger}
}

type slogEventLogger struct {
	logger *slog.Logger
}

func (l *slogEventLogger) Enabled(ctx context.Context, level EventLevel) bool {
	if l == nil || l.logger == nil {
		return false
	}
	return l.logger.Enabled(contextOrBackground(ctx), slogLevelFromEvent(level))
}

func (l *slogEventLogger) LogEvent(ctx context.Context, event Event) {
	if l == nil || l.logger == nil || event == nil {
		return
	}

	ctx = contextOrBackground(ctx)

	switch e := event.(type) {
	case MessageEvent:
		l.logMessage(ctx, e)
	case BuildEvent:
		if e.Err != nil {
			l.logger.Error("app build failed", "app", e.Meta.Name, "profile", e.Profile, "error", e.Err)
			return
		}
		l.logger.Info("app built",
			"app", e.Meta.Name,
			"profile", e.Profile,
			"modules", e.ModuleCount,
			"providers", e.ProviderCount,
			"hooks", e.HookCount,
			"setups", e.SetupCount,
			"invokes", e.InvokeCount,
		)
	case StartEvent:
		if e.Err != nil {
			l.logger.Error("app start failed", "app", e.Meta.Name, "error", e.Err)
			return
		}
		l.logger.Info("app started", "app", e.Meta.Name)
	case StopEvent:
		if e.Err != nil {
			l.logger.Error("app stop failed", "app", e.Meta.Name, "error", e.Err)
			return
		}
		l.logger.Info("app stopped", "app", e.Meta.Name)
	case HealthCheckEvent:
		if e.Err != nil {
			l.logger.Warn("health check failed", "kind", e.Kind, "check", e.Name, "error", e.Err)
			return
		}
		l.logger.Debug("health check passed", "kind", e.Kind, "check", e.Name)
	case StateTransitionEvent:
		l.logger.Debug("runtime state transition",
			"app", e.Meta.Name,
			"from", e.From.String(),
			"to", e.To.String(),
			"reason", e.Reason,
		)
	}
}

func (l *slogEventLogger) logMessage(ctx context.Context, event MessageEvent) {
	args := eventFieldArgs(event.Fields)
	switch event.Level {
	case EventLevelDebug:
		l.logger.DebugContext(ctx, event.Message, args...)
	case EventLevelInfo:
		l.logger.InfoContext(ctx, event.Message, args...)
	case EventLevelWarn:
		l.logger.WarnContext(ctx, event.Message, args...)
	case EventLevelError:
		l.logger.ErrorContext(ctx, event.Message, args...)
	default:
		l.logger.Log(ctx, slogLevelFromEvent(event.Level), event.Message, args...)
	}
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func eventLoggerEnabled(logger EventLogger, ctx context.Context, level EventLevel) bool {
	if logger == nil {
		return false
	}
	if enabler, ok := logger.(eventLoggerEnabler); ok {
		return enabler.Enabled(contextOrBackground(ctx), level)
	}
	return true
}

func emitEventLogger(ctx context.Context, logger EventLogger, event Event) {
	if logger == nil || event == nil {
		return
	}
	logger.LogEvent(contextOrBackground(ctx), event)
}

func logMessageEvent(ctx context.Context, logger EventLogger, level EventLevel, message string, args ...any) {
	if !eventLoggerEnabled(logger, ctx, level) {
		return
	}
	emitEventLogger(ctx, logger, MessageEvent{
		Level:   level,
		Message: message,
		Fields:  eventFields(args...),
	})
}

func eventFields(args ...any) *collectionlist.List[EventField] {
	if len(args) == 0 {
		return collectionlist.NewList[EventField]()
	}

	fields := collectionlist.NewListWithCapacity[EventField]((len(args) + 1) / 2)
	for i := 0; i < len(args); i += 2 {
		key := fmt.Sprintf("arg_%d", i)
		if name, ok := args[i].(string); ok && name != "" {
			key = name
		}

		var value any
		if i+1 < len(args) {
			value = args[i+1]
		}

		fields.Add(EventField{Key: key, Value: value})
	}
	return fields
}

func eventFieldArgs(fields *collectionlist.List[EventField]) []any {
	if fields == nil || fields.Len() == 0 {
		return nil
	}

	values := make([]any, 0, fields.Len()*2)
	fields.Range(func(_ int, field EventField) bool {
		values = append(values, field.Key, field.Value)
		return true
	})
	return values
}

func slogLevelFromEvent(level EventLevel) slog.Level {
	switch level {
	case EventLevelDebug:
		return slog.LevelDebug
	case EventLevelInfo:
		return slog.LevelInfo
	case EventLevelWarn:
		return slog.LevelWarn
	case EventLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (spec *appSpec) resolvedEventLogger() EventLogger {
	if spec == nil {
		return nil
	}
	if spec.eventLogger != nil {
		return spec.eventLogger
	}
	if spec.logger != nil {
		return NewSlogEventLogger(spec.logger)
	}
	return nil
}
