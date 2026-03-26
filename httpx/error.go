package httpx

import (
	"errors"

	"github.com/samber/mo"
)

// Common package-level errors returned by registration and adapter helpers.
var (
	ErrAdapterNotFound    = errors.New("httpx: adapter not found")
	ErrInvalidEndpoint    = errors.New("httpx: invalid endpoint struct")
	ErrInvalidHandlerName = errors.New("httpx: invalid handler function name")
	ErrInvalidHandlerSig  = errors.New("httpx: invalid handler signature")
	ErrRouteAlreadyExists = errors.New("httpx: route already registered")
	ErrRouteNotRegistered = errors.New("httpx: route not registered")
	ErrServerFrozen       = errors.New("httpx: server configuration is frozen")
)

// Error wraps an HTTP status code, message, and optional underlying cause.
type Error struct {
	Code    int
	Message string
	Err     error
}

// Error returns the formatted error message.
func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying cause, if any.
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError constructs an `httpx.Error` with an optional wrapped error.
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

// ToOption converts the error into an optional error value.
func (e *Error) ToOption() mo.Option[error] {
	if e == nil {
		return mo.None[error]()
	}
	return mo.Some[error](e)
}

// IsAdapterNotFound reports whether the error wraps `ErrAdapterNotFound`.
func IsAdapterNotFound(err error) bool {
	return errors.Is(err, ErrAdapterNotFound)
}

// IsInvalidEndpoint reports whether the error wraps `ErrInvalidEndpoint`.
func IsInvalidEndpoint(err error) bool {
	return errors.Is(err, ErrInvalidEndpoint)
}

// IsInvalidHandlerSignature reports whether the error wraps `ErrInvalidHandlerSig`.
func IsInvalidHandlerSignature(err error) bool {
	return errors.Is(err, ErrInvalidHandlerSig)
}

// IsRouteAlreadyRegistered reports whether the error wraps `ErrRouteAlreadyExists`.
func IsRouteAlreadyRegistered(err error) bool {
	return errors.Is(err, ErrRouteAlreadyExists)
}
