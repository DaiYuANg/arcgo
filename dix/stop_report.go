package dix

import (
	"errors"
	"strings"

	do "github.com/samber/do/v2"
	"github.com/samber/lo"
)

type StopReport struct {
	HookError      error
	ShutdownReport *do.ShutdownReport
}

func (r *StopReport) HasErrors() bool {
	return r != nil && r.Err() != nil
}

func (r *StopReport) collectErrors() []error {
	if r == nil {
		return nil
	}
	errs := lo.Compact([]error{r.HookError})
	if r.ShutdownReport != nil && len(r.ShutdownReport.Errors) > 0 {
		errs = append(errs, r.ShutdownReport)
	}
	return errs
}

func (r *StopReport) Err() error {
	return errors.Join(r.collectErrors()...)
}

func (r *StopReport) Error() string {
	errs := r.collectErrors()
	if len(errs) == 0 {
		return ""
	}
	return strings.Join(lo.Map(errs, func(e error, _ int) string { return e.Error() }), "\n")
}
