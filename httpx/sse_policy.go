package httpx

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

// SSERoutePolicy is a typed SSE policy that can affect runtime streaming behavior and OpenAPI output.
type SSERoutePolicy[I any] struct {
	Name string
	// Wrap mutates runtime behavior by wrapping the SSE handler.
	Wrap func(next SSEHandler[I]) SSEHandler[I]
	// Operation mutates OpenAPI operation metadata before registration.
	Operation func(*huma.Operation)
}

// SSEPolicyOperation converts one or more operation options into an SSE route policy.
func SSEPolicyOperation[I any](operationOptions ...OperationOption) SSERoutePolicy[I] {
	return SSERoutePolicy[I]{
		Name:      "operation",
		Operation: buildOperationMutation(operationOptions),
	}
}

func applySSERoutePolicies[I any](handler SSEHandler[I], policies []SSERoutePolicy[I]) SSEHandler[I] {
	if len(policies) == 0 || handler == nil {
		return handler
	}

	return applyWrappers(handler, lo.Map(policies, func(policy SSERoutePolicy[I], _ int) func(SSEHandler[I]) SSEHandler[I] {
		return policy.Wrap
	}))
}

func applySSEPolicyOperations[I any](op *huma.Operation, policies []SSERoutePolicy[I]) {
	applyOperationMutations(op, lo.Map(policies, func(policy SSERoutePolicy[I], _ int) func(*huma.Operation) {
		return policy.Operation
	}))
}
