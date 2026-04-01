package clientx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
)

var backgroundContext = context.Background()

// OperationKind classifies the kind of client operation being executed.
type OperationKind string

const (
	// OperationKindUnknown indicates that the operation kind is not known.
	OperationKindUnknown OperationKind = "unknown"
	// OperationKindRequest identifies application-layer request execution.
	OperationKindRequest OperationKind = "request"
	// OperationKindDial identifies outbound connection establishment.
	OperationKindDial OperationKind = "dial"
	// OperationKindListen identifies local packet listener setup.
	OperationKindListen OperationKind = "listen"
)

// Operation describes the client operation visible to policies and hooks.
type Operation struct {
	Protocol Protocol
	Kind     OperationKind
	Op       string
	Network  string
	Addr     string
}

// Policy hooks into operation execution before and after the transport call.
type Policy interface {
	Before(ctx context.Context, operation Operation) (context.Context, error)
	After(ctx context.Context, operation Operation, err error) error
}

// RetryDecider allows a policy to request re-execution with an optional delay.
type RetryDecider interface {
	ShouldRetry(ctx context.Context, operation Operation, attempt int, err error) (retry bool, wait time.Duration)
}

// PolicyFuncs adapts plain functions to the Policy interface.
type PolicyFuncs struct {
	BeforeFunc func(ctx context.Context, operation Operation) (context.Context, error)
	AfterFunc  func(ctx context.Context, operation Operation, err error) error
}

// Before dispatches to BeforeFunc when configured.
func (p PolicyFuncs) Before(ctx context.Context, operation Operation) (context.Context, error) {
	if p.BeforeFunc != nil {
		return p.BeforeFunc(ctx, operation)
	}
	return ctx, nil
}

// After dispatches to AfterFunc when configured.
func (p PolicyFuncs) After(ctx context.Context, operation Operation, err error) error {
	if p.AfterFunc != nil {
		return p.AfterFunc(ctx, operation, err)
	}
	return nil
}

// InvokeWithPolicies executes fn with the configured policy chain and retry semantics.
func InvokeWithPolicies[T any](
	ctx context.Context,
	operation Operation,
	fn func(context.Context) (T, error),
	policies ...Policy,
) (T, error) {
	var zero T
	ctx = normalizeContext(ctx)
	if fn == nil {
		return zero, errors.New("invoke function is nil")
	}
	operation = normalizeOperation(operation)

	activePolicies := filterPolicies(policies)

	for attempt := 1; ; attempt++ {
		attemptCtx, applied, err := applyBeforePolicies(ctx, activePolicies, operation)
		if err != nil {
			return zero, applyAfterPolicies(attemptCtx, applied, operation, err)
		}

		result, execErr := fn(attemptCtx)
		execErr = applyAfterPolicies(attemptCtx, applied, operation, execErr)
		if execErr == nil {
			return result, nil
		}

		retry, wait := decideRetry(ctx, activePolicies, operation, attempt, execErr)
		if !retry {
			return result, execErr
		}
		if sleepErr := sleepWithContext(ctx, wait); sleepErr != nil {
			return result, errors.Join(execErr, sleepErr)
		}
	}
}

func applyAfterPolicies(ctx context.Context, policies []Policy, operation Operation, baseErr error) error {
	aggErr := baseErr
	for index := len(policies) - 1; index >= 0; index-- {
		afterOK, afterErr := callPolicyAfter(ctx, policies[index], operation, aggErr)
		if afterOK && afterErr != nil {
			aggErr = errors.Join(aggErr, afterErr)
		}
	}
	return aggErr
}

func decideRetry(
	ctx context.Context,
	policies []Policy,
	operation Operation,
	attempt int,
	err error,
) (retry bool, wait time.Duration) {
	type retryDecision struct {
		retry bool
		wait  time.Duration
	}

	decision := lo.Reduce(policies, func(agg retryDecision, policy Policy, _ int) retryDecision {
		decider, ok := policy.(RetryDecider)
		if !ok {
			return agg
		}
		shouldRetry, delay, retryOK := callShouldRetry(ctx, decider, operation, attempt, err)
		if !retryOK || !shouldRetry {
			return agg
		}

		agg.retry = true
		agg.wait = max(agg.wait, delay)
		return agg
	}, retryDecision{})

	return decision.retry, max(decision.wait, 0)
}

func filterPolicies(policies []Policy) []Policy {
	return lo.Filter(policies, func(policy Policy, _ int) bool {
		return policy != nil
	})
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return backgroundContext
	}
	return ctx
}

func normalizeOperation(operation Operation) Operation {
	if operation.Protocol == "" {
		operation.Protocol = ProtocolUnknown
	}
	if operation.Kind == "" {
		operation.Kind = OperationKindUnknown
	}
	return operation
}

func applyBeforePolicies(
	ctx context.Context,
	policies []Policy,
	operation Operation,
) (context.Context, []Policy, error) {
	if len(policies) == 0 {
		return ctx, nil, nil
	}

	nextCtx, err := callPolicyBefore(ctx, policies[0], operation)
	if err != nil {
		return ctx, nil, err
	}

	finalCtx, tailApplied, tailErr := applyBeforePolicies(nextCtx, policies[1:], operation)
	if tailErr != nil {
		return finalCtx, append([]Policy{policies[0]}, tailApplied...), tailErr
	}

	return finalCtx, append([]Policy{policies[0]}, tailApplied...), nil
}

func callPolicyBefore(
	ctx context.Context,
	policy Policy,
	operation Operation,
) (context.Context, error) {
	recovered := false
	defer func() {
		if recover() != nil {
			recovered = true
		}
	}()

	policyCtx, policyErr := policy.Before(ctx, operation)
	if recovered {
		return ctx, nil
	}
	if policyCtx == nil {
		return ctx, wrapPolicyBeforeError(policyErr)
	}
	return policyCtx, wrapPolicyBeforeError(policyErr)
}

func callPolicyAfter(
	ctx context.Context,
	policy Policy,
	operation Operation,
	err error,
) (ok bool, afterErr error) {
	ok = true
	defer func() {
		if recover() != nil {
			ok = false
			afterErr = nil
		}
	}()
	return true, wrapPolicyAfterError(policy.After(ctx, operation, err))
}

func callShouldRetry(
	ctx context.Context,
	decider RetryDecider,
	operation Operation,
	attempt int,
	err error,
) (retry bool, wait time.Duration, ok bool) {
	ok = true
	defer func() {
		if recover() != nil {
			retry = false
			wait = 0
			ok = false
		}
	}()
	retry, wait = decider.ShouldRetry(ctx, operation, attempt, err)
	return retry, wait, ok
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context done: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func wrapPolicyBeforeError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("policy before: %w", err)
}

func wrapPolicyAfterError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("policy after: %w", err)
}
