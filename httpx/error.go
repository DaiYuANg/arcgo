package httpx

import (
	"errors"

	"github.com/samber/mo"
)

// 常见错误
var (
	ErrAdapterNotFound    = errors.New("httpx: adapter not found")
	ErrInvalidEndpoint    = errors.New("httpx: invalid endpoint struct")
	ErrInvalidHandlerName = errors.New("httpx: invalid handler function name")
	ErrInvalidHandlerSig  = errors.New("httpx: invalid handler signature")
	ErrRouteNotRegistered = errors.New("httpx: route not registered")
)

// Error httpx 错误类型
type Error struct {
	Code    int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// NewError 创建 httpx 错误
func NewError(code int, message string, err ...error) *Error {
	e := &Error{
		Code:    code,
		Message: message,
	}
	if len(err) > 0 {
		e.Err = err[0]
	}
	return e
}

// ToOption 将 Error 转换为 Option
func (e *Error) ToOption() mo.Option[error] {
	if e == nil {
		return mo.None[error]()
	}
	return mo.Some[error](e)
}

// IsAdapterNotFound 检查错误是否为 ErrAdapterNotFound
func IsAdapterNotFound(err error) bool {
	return errors.Is(err, ErrAdapterNotFound)
}

// IsInvalidEndpoint 检查错误是否为 ErrInvalidEndpoint
func IsInvalidEndpoint(err error) bool {
	return errors.Is(err, ErrInvalidEndpoint)
}

// IsInvalidHandlerSignature 检查错误是否为 ErrInvalidHandlerSig
func IsInvalidHandlerSignature(err error) bool {
	return errors.Is(err, ErrInvalidHandlerSig)
}
