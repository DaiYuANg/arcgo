package dix

import (
	"context"
	"sync"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/oops"
)

type appPlanCache struct {
	once   sync.Once
	plan   *buildPlan
	report ValidationReport
	err    error
}

func (a *App) cachedBuildPlan(ctx context.Context) (*buildPlan, ValidationReport, error) {
	if a == nil || a.spec == nil {
		err := oops.In("dix").
			With("op", "cached_build_plan").
			New("app is nil")
		return nil, ValidationReport{Errors: collectionx.NewList(err)}, err
	}

	a.planCache.once.Do(func() {
		a.planCache.plan, a.planCache.report, a.planCache.err = computeBuildPlan(ctx, a)
	})

	return a.planCache.plan, cloneValidationReport(a.planCache.report), a.planCache.err
}

func computeBuildPlan(ctx context.Context, app *App) (*buildPlan, ValidationReport, error) {
	plan, err := newUnvalidatedBuildPlan(ctx, app)
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
