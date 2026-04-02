package dix

import (
	"errors"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

// Validate validates the immutable app spec and current module graph.
func (a *App) Validate() error {
	return a.ValidateReport().Err()
}

// ValidateReport validates the app and returns the full validation report.
func (a *App) ValidateReport() ValidationReport {
	plan, err := newBuildPlan(a)
	if err != nil {
		return ValidationReport{Errors: collectionx.NewList(err)}
	}
	return validateTypedGraphReport(plan)
}

// HasWarnings reports whether the validation report contains warnings.
func (r ValidationReport) HasWarnings() bool {
	return r.Warnings != nil && r.Warnings.Len() > 0
}

// HasErrors reports whether the validation report contains errors.
func (r ValidationReport) HasErrors() bool {
	return r.Errors != nil && r.Errors.Len() > 0
}

// Err returns the combined validation error.
func (r ValidationReport) Err() error {
	if r.Errors == nil {
		return nil
	}
	return errors.Join(r.Errors.Values()...)
}

// WarningSummary renders the validation warnings as a newline-delimited summary.
func (r ValidationReport) WarningSummary() string {
	if r.Warnings == nil || r.Warnings.Len() == 0 {
		return ""
	}

	return strings.Join(lo.Map(r.Warnings.Values(), func(warning ValidationWarning, _ int) string {
		line := string(warning.Kind)
		if warning.Module != "" {
			line += " module=" + warning.Module
		}
		if warning.Label != "" {
			line += " label=" + warning.Label
		}
		if warning.Details != "" {
			line += " " + warning.Details
		}
		return line
	}), "\n")
}
