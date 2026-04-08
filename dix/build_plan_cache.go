package dix

import (
	"errors"
	"sync"

	"github.com/DaiYuANg/arcgo/collectionx"
)

type appPlanCache struct {
	once   sync.Once
	plan   *buildPlan
	report ValidationReport
	err    error
}

func (a *App) cachedBuildPlan() (*buildPlan, ValidationReport, error) {
	if a == nil || a.spec == nil {
		err := errors.New("app is nil")
		return nil, ValidationReport{Errors: collectionx.NewList(err)}, err
	}

	a.planCache.once.Do(func() {
		a.planCache.plan, a.planCache.report, a.planCache.err = computeBuildPlan(a)
	})

	return a.planCache.plan, cloneValidationReport(a.planCache.report), a.planCache.err
}

func computeBuildPlan(app *App) (*buildPlan, ValidationReport, error) {
	plan, err := newUnvalidatedBuildPlan(app)
	if err != nil {
		report := ValidationReport{Errors: collectionx.NewList(err)}
		return nil, report, err
	}

	report := validateTypedGraphReport(plan)
	if reportErr := report.Err(); reportErr != nil {
		return plan, report, reportErr
	}

	return plan, report, nil
}
