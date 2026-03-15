package httpx

import "github.com/DaiYuANg/archgo/httpx/adapter"

// UseAdapter runs `use` with the underlying adapter capability when available.
// It returns true when the capability exists and `use` was called.
func UseAdapter[T any](s ServerRuntime, use func(T)) bool {
	server := unwrapServer(s)
	if server == nil || server.adapter == nil || use == nil {
		return false
	}
	return withAdapterCapability(server.adapter, use)
}

func withAdapterCapability[T any](a adapter.Adapter, use func(T)) bool {
	if a == nil || use == nil {
		return false
	}

	capability, ok := any(a).(T)
	if !ok {
		return false
	}
	use(capability)
	return true
}
