package httpx

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

// RoutePolicy is a typed policy that can affect both runtime handling and OpenAPI output.
type RoutePolicy[I, O any] struct {
	Name string
	// Wrap mutates runtime behavior by wrapping the route handler.
	Wrap func(next TypedHandler[I, O]) TypedHandler[I, O]
	// Operation mutates OpenAPI operation metadata before registration.
	Operation func(*huma.Operation)
}

// PolicyOperation converts one or more operation options into a route policy.
func PolicyOperation[I, O any](operationOptions ...OperationOption) RoutePolicy[I, O] {
	return RoutePolicy[I, O]{
		Name:      "operation",
		Operation: buildOperationMutation(operationOptions),
	}
}

func applyRoutePolicies[I, O any](handler TypedHandler[I, O], policies []RoutePolicy[I, O]) TypedHandler[I, O] {
	if len(policies) == 0 || handler == nil {
		return handler
	}

	return applyWrappers(handler, lo.Map(policies, func(policy RoutePolicy[I, O], _ int) func(TypedHandler[I, O]) TypedHandler[I, O] {
		return policy.Wrap
	}))
}

func applyPolicyOperations[I, O any](op *huma.Operation, policies []RoutePolicy[I, O]) {
	applyOperationMutations(op, lo.Map(policies, func(policy RoutePolicy[I, O], _ int) func(*huma.Operation) {
		return policy.Operation
	}))
}
