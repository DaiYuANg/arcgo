package repository

import "errors"

var ErrNotFound = errors.New("dbx/repository: not found")
var ErrNilMutation = errors.New("dbx/repository: mutation query is nil")
var ErrConflict = errors.New("dbx/repository: conflict")
var ErrValidation = errors.New("dbx/repository: validation")
var ErrVersionConflict = errors.New("dbx/repository: version conflict")

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	if e.Message == "" {
		return "dbx/repository: validation failed"
	}
	return "dbx/repository: validation failed: " + e.Message
}
func (e *ValidationError) Unwrap() error { return ErrValidation }

type ConflictError struct{ Err error }

func (e *ConflictError) Error() string { return e.Err.Error() }
func (e *ConflictError) Unwrap() error { return ErrConflict }

type VersionConflictError struct{ Err error }

func (e *VersionConflictError) Error() string { return e.Err.Error() }
func (e *VersionConflictError) Unwrap() error { return ErrVersionConflict }

