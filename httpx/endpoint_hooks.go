package httpx

import "github.com/samber/lo"

func runEndpointHooks(server ServerRuntime, endpoint Endpoint, hooks []EndpointHooks, selectHook func(EndpointHooks) EndpointHookFunc) {
	lo.ForEach(lo.FilterMap(hooks, func(hook EndpointHooks, _ int) (EndpointHookFunc, bool) {
		selected := selectHook(hook)
		return selected, selected != nil
	}), func(hook EndpointHookFunc, _ int) {
		hook(server, endpoint)
	})
}
