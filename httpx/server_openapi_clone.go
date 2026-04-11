package httpx

import (
	"maps"

	"github.com/DaiYuANg/arcgo/collectionx"
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

func expandTags(tags OpenAPITags) []string {
	if tags.IsEmpty() {
		return nil
	}
	return collectionx.FilterList(tags, func(_ int, tag string) bool {
		return tag != ""
	}).Values()
}

func expandTagDefinitions(tags OpenAPITagDefinitions) []*huma.Tag {
	if tags.IsEmpty() {
		return nil
	}
	return collectionx.FilterMapList(tags, func(_ int, tag *huma.Tag) (*huma.Tag, bool) {
		if tag == nil {
			return nil, false
		}
		return cloneTag(tag), true
	}).Values()
}

func expandParameters(params OpenAPIParameters) []*huma.Param {
	if params.IsEmpty() {
		return nil
	}
	return collectionx.FilterMapList(params, func(_ int, param *huma.Param) (*huma.Param, bool) {
		if param == nil {
			return nil, false
		}
		return cloneParam(param), true
	}).Values()
}

func expandExtensions(values OpenAPIExtensions) map[string]any {
	if values.IsEmpty() {
		return nil
	}
	return cloneExtensions(values.All())
}

func expandSecuritySchemes(schemes OpenAPISecuritySchemes) []lo.Entry[string, *huma.SecurityScheme] {
	if schemes.IsEmpty() {
		return nil
	}
	entries := collectionx.NewListWithCapacity[lo.Entry[string, *huma.SecurityScheme]](schemes.Len())
	schemes.Range(func(name string, scheme *huma.SecurityScheme) bool {
		if name != "" && scheme != nil {
			entries.Add(lo.Entry[string, *huma.SecurityScheme]{
				Key:   name,
				Value: scheme,
			})
		}
		return true
	})
	return entries.Values()
}

func expandSecurityRequirements(requirements OpenAPISecurityRequirements) []map[string][]string {
	if requirements.IsEmpty() {
		return nil
	}
	return collectionx.FilterMapList(requirements, func(_ int, req OpenAPISecurityRequirement) (map[string][]string, bool) {
		expanded := expandSecurityRequirement(req)
		return expanded, len(expanded) > 0
	}).Values()
}

func expandSecurityRequirement(requirement OpenAPISecurityRequirement) map[string][]string {
	if requirement.IsEmpty() {
		return nil
	}
	expanded := make(map[string][]string, requirement.Len())
	requirement.Range(func(name string, scopes OpenAPISecurityScopes) bool {
		if name == "" {
			return true
		}
		expanded[name] = expandSecurityScopes(scopes)
		return true
	})
	if len(expanded) == 0 {
		return nil
	}
	return expanded
}

func expandSecurityScopes(scopes OpenAPISecurityScopes) []string {
	if scopes.IsEmpty() {
		return []string{}
	}
	return scopes.Values()
}

func cloneBuiltInSecurityRequirements(requirements []map[string][]string) []map[string][]string {
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
	if len(values) == 0 {
		return nil
	}
	return lo.MapValues(values, func(scopes []string, _ string) []string {
		if len(scopes) == 0 {
			return []string{}
		}
		return append([]string(nil), scopes...)
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
