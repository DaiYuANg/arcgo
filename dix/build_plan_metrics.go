package dix

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
)

func countModuleProviders(modules *collectionlist.List[*moduleSpec]) int {
	total := 0
	if modules == nil {
		return total
	}
	modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod != nil {
			total += mod.providers.Len()
		}
		return true
	})
	return total
}

func countModuleHooks(modules *collectionlist.List[*moduleSpec]) int {
	total := 0
	if modules == nil {
		return total
	}
	modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod != nil {
			total += mod.hooks.Len()
		}
		return true
	})
	return total
}

func countModuleSetups(modules *collectionlist.List[*moduleSpec]) int {
	total := 0
	if modules == nil {
		return total
	}
	modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod != nil {
			total += mod.setups.Len()
		}
		return true
	})
	return total
}

func countModuleInvokes(modules *collectionlist.List[*moduleSpec]) int {
	total := 0
	if modules == nil {
		return total
	}
	modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod != nil {
			total += mod.invokes.Len()
		}
		return true
	})
	return total
}

func serviceRefNames(refs collectionx.List[ServiceRef]) collectionx.List[string] {
	if refs == nil || refs.Len() == 0 {
		return collectionx.NewList[string]()
	}
	return collectionx.FilterMapList(refs, func(_ int, ref ServiceRef) (string, bool) {
		return ref.Name, ref.Name != ""
	})
}
