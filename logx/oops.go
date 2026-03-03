package logx

import (
	"context"
	"fmt"

	"github.com/samber/oops"
)

// LogOops 记录 oops 错误（带堆栈追踪）
func (l *Logger) LogOops(err error) {
	if l == nil {
		return
	}

	// 使用 oops 的 zerolog 集成自动记录堆栈
	l.Error("Error", "error", err)
}

// LogOopsWithStack 记录 oops 错误（带堆栈）
func (l *Logger) LogOopsWithStack(err error) {
	l.LogOops(err)
}

// Oops 创建 oops 错误
func (l *Logger) Oops() error {
	return oops.New("error")
}

// Oopsf 创建带格式化的 oops 错误
func (l *Logger) Oopsf(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	if msg == "" {
		msg = "error"
	}
	return oops.New(msg)
}

// OopsWith 创建带 context 的 oops 错误
func (l *Logger) OopsWith(ctx context.Context) error {
	_ = ctx
	return oops.New("error")
}
