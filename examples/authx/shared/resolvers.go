package shared

import (
	"errors"
	"sort"
	"strings"

	"github.com/DaiYuANg/archgo/collectionx"
	"github.com/samber/lo"
)

type MethodActionResolver struct {
	actionByMethod collectionx.Map[string, string]
}

func NewMethodActionResolver(actionByMethod map[string]string) MethodActionResolver {
	normalized := make(map[string]string, len(actionByMethod))
	lo.ForEach(lo.Entries(actionByMethod), func(entry lo.Entry[string, string], _ int) {
		key := strings.ToUpper(strings.TrimSpace(entry.Key))
		value := strings.TrimSpace(entry.Value)
		if key != "" && value != "" {
			normalized[key] = value
		}
	})

	return MethodActionResolver{
		actionByMethod: collectionx.NewMapFrom(normalized),
	}
}

func (resolver MethodActionResolver) Resolve(method string) (string, error) {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	if action, ok := resolver.actionByMethod.GetOption(normalizedMethod).Get(); ok {
		return action, nil
	}
	return "", errors.New("unsupported method for action mapping")
}

type RouteResourceResolver struct {
	resourceByExactPattern collectionx.Map[string, string]
	resourceByPrefix       collectionx.Map[string, string]
}

func NewRouteResourceResolver(
	resourceByExactPattern map[string]string,
	resourceByPrefix map[string]string,
) RouteResourceResolver {
	return RouteResourceResolver{
		resourceByExactPattern: collectionx.NewMapFrom(normalizedEntries(resourceByExactPattern)),
		resourceByPrefix:       collectionx.NewMapFrom(normalizedEntries(resourceByPrefix)),
	}
}

func (resolver RouteResourceResolver) Resolve(routePattern string) (string, error) {
	pattern := strings.TrimSpace(routePattern)
	if pattern == "" {
		return "", errors.New("empty route pattern")
	}

	if resource, ok := resolver.resourceByExactPattern.GetOption(pattern).Get(); ok {
		return resource, nil
	}

	prefixes := resolver.resourceByPrefix.Keys()
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i]) > len(prefixes[j])
	})
	if prefix, found := lo.Find(prefixes, func(p string) bool {
		return strings.HasPrefix(pattern, p)
	}); found {
		if resource, ok := resolver.resourceByPrefix.GetOption(prefix).Get(); ok {
			return resource, nil
		}
	}

	return "", errors.New("unsupported route pattern for resource mapping")
}

func normalizedEntries(entries map[string]string) map[string]string {
	if len(entries) == 0 {
		return nil
	}

	normalized := make(map[string]string, len(entries))
	lo.ForEach(lo.Entries(entries), func(entry lo.Entry[string, string], _ int) {
		key := strings.TrimSpace(entry.Key)
		value := strings.TrimSpace(entry.Value)
		if key != "" && value != "" {
			normalized[key] = value
		}
	})

	return normalized
}
