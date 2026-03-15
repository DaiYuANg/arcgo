package httpx

import (
	"strings"

	"github.com/DaiYuANg/archgo/collectionx/set"
	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

// Group represents a route group backed by a Huma group when available.
type Group struct {
	server    *Server
	prefix    string
	humaGroup *huma.Group
}

// Group creates a prefixed route group under the server base path.
func (s *Server) Group(prefix string) *Group {
	normalizedPrefix := normalizeRoutePrefix(prefix)
	var humaGroup *huma.Group
	if api := s.HumaAPI(); api != nil {
		humaGroup = huma.NewGroup(api, joinRoutePath(s.basePath, normalizedPrefix))
	}
	return &Group{
		server:    s,
		prefix:    normalizedPrefix,
		humaGroup: humaGroup,
	}
}

// HumaGroup exposes the underlying Huma group when one is available.
func (g *Group) HumaGroup() *huma.Group {
	if g == nil {
		return nil
	}
	return g.humaGroup
}

// UseHumaMiddleware registers Huma middleware on the group.
func (g *Group) UseHumaMiddleware(middlewares ...func(huma.Context, func(huma.Context))) {
	if g == nil || g.humaGroup == nil || len(middlewares) == 0 {
		return
	}
	if !g.server.allowConfigMutation("Group.UseHumaMiddleware") {
		return
	}
	g.humaGroup.UseMiddleware(middlewares...)
}

// UseOperationModifier registers a Huma operation modifier on the group.
func (g *Group) UseOperationModifier(modifier func(*huma.Operation, func(*huma.Operation))) {
	if g == nil || g.humaGroup == nil || modifier == nil {
		return
	}
	if !g.server.allowConfigMutation("Group.UseOperationModifier") {
		return
	}
	g.humaGroup.UseModifier(modifier)
}

// UseSimpleOperationModifier registers a simple operation modifier on the group.
func (g *Group) UseSimpleOperationModifier(modifier func(*huma.Operation)) {
	if g == nil || g.humaGroup == nil || modifier == nil {
		return
	}
	if !g.server.allowConfigMutation("Group.UseSimpleOperationModifier") {
		return
	}
	g.humaGroup.UseSimpleModifier(modifier)
}

// UseResponseTransformer registers response transformers on the group.
func (g *Group) UseResponseTransformer(transformers ...huma.Transformer) {
	if g == nil || g.humaGroup == nil || len(transformers) == 0 {
		return
	}
	if !g.server.allowConfigMutation("Group.UseResponseTransformer") {
		return
	}
	g.humaGroup.UseTransformer(transformers...)
}

// DefaultTags applies group-level default tags to future operations.
func (g *Group) DefaultTags(tags ...string) {
	if g == nil || g.humaGroup == nil || len(tags) == 0 {
		return
	}
	if !g.server.allowConfigMutation("Group.DefaultTags") {
		return
	}
	g.humaGroup.UseSimpleModifier(func(op *huma.Operation) {
		existing := set.NewSet(op.Tags...)
		newTags := lo.Filter(tags, func(tag string, _ int) bool {
			return tag != "" && !existing.Contains(tag)
		})
		op.Tags = append(op.Tags, newTags...)
	})
}

// DefaultSecurity applies group-level default security to operations that do not override it.
func (g *Group) DefaultSecurity(requirements ...map[string][]string) {
	if g == nil || g.humaGroup == nil || len(requirements) == 0 {
		return
	}
	if !g.server.allowConfigMutation("Group.DefaultSecurity") {
		return
	}
	cloned := cloneSecurityRequirements(requirements)
	g.humaGroup.UseSimpleModifier(func(op *huma.Operation) {
		if len(op.Security) == 0 {
			op.Security = cloneSecurityRequirements(cloned)
		}
	})
}

// DefaultParameters applies group-level parameters to future operations.
func (g *Group) DefaultParameters(params ...*huma.Param) {
	if g == nil || g.humaGroup == nil || len(params) == 0 {
		return
	}
	if !g.server.allowConfigMutation("Group.DefaultParameters") {
		return
	}
	cloned := lo.FilterMap(params, func(param *huma.Param, _ int) (*huma.Param, bool) {
		if param == nil {
			return nil, false
		}
		return cloneParam(param), true
	})
	if len(cloned) == 0 {
		return
	}
	g.humaGroup.UseSimpleModifier(func(op *huma.Operation) {
		lo.ForEach(cloned, func(param *huma.Param, _ int) {
			appendOperationParameter(op, param)
		})
	})
}

// DefaultSummaryPrefix prepends a group-level summary prefix to future operations.
func (g *Group) DefaultSummaryPrefix(prefix string) {
	if g == nil || g.humaGroup == nil || strings.TrimSpace(prefix) == "" {
		return
	}
	if !g.server.allowConfigMutation("Group.DefaultSummaryPrefix") {
		return
	}
	trimmed := strings.TrimSpace(prefix)
	g.humaGroup.UseSimpleModifier(func(op *huma.Operation) {
		if strings.TrimSpace(op.Summary) == "" {
			op.Summary = trimmed
			return
		}
		if strings.HasPrefix(op.Summary, trimmed) {
			return
		}
		op.Summary = trimmed + " " + op.Summary
	})
}

// DefaultDescription applies a group-level description when an operation does not define one.
func (g *Group) DefaultDescription(description string) {
	if g == nil || g.humaGroup == nil || strings.TrimSpace(description) == "" {
		return
	}
	if !g.server.allowConfigMutation("Group.DefaultDescription") {
		return
	}
	trimmed := strings.TrimSpace(description)
	g.humaGroup.UseSimpleModifier(func(op *huma.Operation) {
		if strings.TrimSpace(op.Description) == "" {
			op.Description = trimmed
		}
	})
}

// RegisterTags adds OpenAPI tag metadata for this group context.
func (g *Group) RegisterTags(tags ...*huma.Tag) {
	if g == nil || g.server == nil || len(tags) == 0 {
		return
	}
	if !g.server.allowConfigMutation("Group.RegisterTags") {
		return
	}
	lo.ForEach(tags, func(tag *huma.Tag, _ int) {
		if tag != nil {
			g.server.AddTag(tag)
		}
	})
}

// DefaultExternalDocs applies group-level external docs to future operations
// when an operation does not define its own external docs.
func (g *Group) DefaultExternalDocs(docs *huma.ExternalDocs) {
	if g == nil || g.humaGroup == nil || docs == nil {
		return
	}
	if !g.server.allowConfigMutation("Group.DefaultExternalDocs") {
		return
	}
	cloned := cloneExternalDocs(docs)
	g.humaGroup.UseSimpleModifier(func(op *huma.Operation) {
		if op.ExternalDocs == nil {
			op.ExternalDocs = cloneExternalDocs(cloned)
		}
	})
}

// DefaultExtensions applies group-level OpenAPI extensions to future operations.
func (g *Group) DefaultExtensions(extensions map[string]any) {
	if g == nil || g.humaGroup == nil || len(extensions) == 0 {
		return
	}
	if !g.server.allowConfigMutation("Group.DefaultExtensions") {
		return
	}
	cloned := cloneExtensions(extensions)
	g.humaGroup.UseSimpleModifier(func(op *huma.Operation) {
		if op.Extensions == nil {
			op.Extensions = map[string]any{}
		}
		lo.ForEach(lo.Entries(cloned), func(entry lo.Entry[string, any], _ int) {
			if _, exists := op.Extensions[entry.Key]; !exists {
				op.Extensions[entry.Key] = entry.Value
			}
		})
	})
}
