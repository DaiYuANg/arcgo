package dix

import (
	"errors"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/do/v2"
)

// StopReport aggregates errors produced while stopping a runtime.
type StopReport struct {
	HookError      error
	ShutdownReport *do.ShutdownReport
}

// HasErrors reports whether the stop report contains any errors.
func (r *StopReport) HasErrors() bool {
	return r != nil && r.Err() != nil
}

func (r *StopReport) collectErrors() collectionx.List[error] {
	if r == nil {
		return collectionx.NewList[error]()
	}
	errs := collectionx.NewListWithCapacity[error](2)
	if r.HookError != nil {
		errs.Add(r.HookError)
	}
	if r.ShutdownReport != nil && len(r.ShutdownReport.Errors) > 0 {
		errs.Add(r.ShutdownReport)
	}
	return errs
}

// Err returns the combined stop error.
func (r *StopReport) Err() error {
	return errors.Join(r.collectErrors().Values()...)
}

// Error returns the combined stop error string.
func (r *StopReport) Error() string {
	errs := r.collectErrors()
	if errs.Len() == 0 {
		return ""
	}
	lines := collectionx.MapList(errs, func(_ int, err error) string {
		return err.Error()
	})
	return strings.Join(lines.Values(), "\n")
}
