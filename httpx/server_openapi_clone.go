package httpx

import (
	"maps"

	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

func cloneParam(param *huma.Param) *huma.Param {
	if param == nil {
		return nil
	}
	cloned := *param
	if param.Schema != nil {
		cloned.Schema = new(*param.Schema)
	}
	if param.Examples != nil {
		cloned.Examples = make(map[string]*huma.Example, len(param.Examples))
		maps.Copy(cloned.Examples, param.Examples)
	}
	if param.Extensions != nil {
		cloned.Extensions = make(map[string]any, len(param.Extensions))
		maps.Copy(cloned.Extensions, param.Extensions)
	}
	return &cloned
}

func cloneTag(tag *huma.Tag) *huma.Tag {
	if tag == nil {
		return nil
	}
	cloned := *tag
	if tag.Extensions != nil {
		cloned.Extensions = make(map[string]any, len(tag.Extensions))
		maps.Copy(cloned.Extensions, tag.Extensions)
	}
	return &cloned
}

func cloneExternalDocs(docs *huma.ExternalDocs) *huma.ExternalDocs {
	if docs == nil {
		return nil
	}
	cloned := *docs
	cloned.Extensions = cloneExtensions(docs.Extensions)
	return &cloned
}

func cloneSecurityScheme(scheme *huma.SecurityScheme) *huma.SecurityScheme {
	if scheme == nil {
		return nil
	}
	cloned := *scheme
	if scheme.Extensions != nil {
		cloned.Extensions = make(map[string]any, len(scheme.Extensions))
		maps.Copy(cloned.Extensions, scheme.Extensions)
	}
	return &cloned
}

func cloneExtensions(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	maps.Copy(cloned, values)
	return cloned
}

func cloneSecurityRequirements(requirements []map[string][]string) []map[string][]string {
	if len(requirements) == 0 {
		return nil
	}
	return lo.Map(requirements, func(req map[string][]string, _ int) map[string][]string {
		if req == nil {
			return nil
		}
		return cloneStringSliceMap(req)
	})
}

func cloneStringSliceMap(values map[string][]string) map[string][]string {
	return lo.MapValues(values, func(scopes []string, _ string) []string {
		if scopes == nil {
			return []string{}
		}
		return lo.Map(scopes, func(scope string, _ int) string {
			return scope
		})
	})
}

func findTag(tags []*huma.Tag, name string) int {
	_, index, ok := lo.FindIndexOf(tags, func(tag *huma.Tag) bool {
		return tag != nil && tag.Name == name
	})
	if !ok {
		return -1
	}
	return index
}
