package dbx

import "log/slog"

type runtimeTraceSession interface {
	Logger() *slog.Logger
	Debug() bool
}

func logRuntimeNode(session Session, node string, attrs ...any) {
	if session == nil {
		return
	}
	traced, ok := session.(runtimeTraceSession)
	if !ok || traced == nil || !traced.Debug() || traced.Logger() == nil {
		return
	}
	fields := make([]any, 0, len(attrs)+2)
	fields = append(fields, "node", node)
	fields = append(fields, attrs...)
	traced.Logger().Debug("dbx runtime node", fields...)
}
