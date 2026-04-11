package httpx

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

// ConfigureOpenAPI mutates the underlying Huma OpenAPI document.
func (s *Server) ConfigureOpenAPI(fn func(*huma.OpenAPI)) {
	if fn == nil {
		return
	}
	if !s.allowConfigMutation("ConfigureOpenAPI") {
		return
	}
	openAPI := s.OpenAPI()
	if openAPI == nil {
		return
	}
	s.openAPIMu.Lock()
	defer s.openAPIMu.Unlock()
	fn(openAPI)
}

// PatchOpenAPI mutates the underlying Huma OpenAPI document.
func (s *Server) PatchOpenAPI(fn func(*huma.OpenAPI)) {
	s.ConfigureOpenAPI(fn)
}

// UseOpenAPIPatch appends an OpenAPI patch and applies it immediately.
func (s *Server) UseOpenAPIPatch(fn func(*huma.OpenAPI)) {
	if s == nil || fn == nil {
		return
	}
	if !s.allowConfigMutation("UseOpenAPIPatch") {
		return
	}
	s.openAPIPatches.Add(fn)
	s.ConfigureOpenAPI(fn)
}

// UseHumaMiddleware registers API-level Huma middleware.
func (s *Server) UseHumaMiddleware(middlewares ...func(huma.Context, func(huma.Context))) {
	if len(middlewares) == 0 {
		return
	}
	if !s.allowConfigMutation("UseHumaMiddleware") {
		return
	}
	s.humaMiddlewares.Add(middlewares...)
	if api := s.HumaAPI(); api != nil {
		api.UseMiddleware(middlewares...)
	}
}

// UseOperationModifier registers a server-level operation modifier for future operations.
func (s *Server) UseOperationModifier(modifier func(*huma.Operation)) {
	if s == nil || modifier == nil {
		return
	}
	if !s.allowConfigMutation("UseOperationModifier") {
		return
	}
	s.operationModifiers.Add(modifier)
}

// AddTag registers OpenAPI tag metadata.
func (s *Server) AddTag(tag *huma.Tag) {
	if s == nil || tag == nil || tag.Name == "" {
		return
	}
	if !s.allowConfigMutation("AddTag") {
		return
	}
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		cloned := cloneTag(tag)
		if index := findTag(doc.Tags, tag.Name); index >= 0 {
			doc.Tags[index] = cloned
			return
		}
		doc.Tags = lo.Concat(doc.Tags, []*huma.Tag{cloned})
	})
}

// RegisterSecurityScheme registers an OpenAPI security scheme component.
func (s *Server) RegisterSecurityScheme(name string, scheme *huma.SecurityScheme) {
	if s == nil || name == "" || scheme == nil {
		return
	}
	if !s.allowConfigMutation("RegisterSecurityScheme") {
		return
	}
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		components := ensureComponents(doc)
		if components.SecuritySchemes == nil {
			components.SecuritySchemes = map[string]*huma.SecurityScheme{}
		}
		components.SecuritySchemes[name] = cloneSecurityScheme(scheme)
	})
}

// SetDefaultSecurity configures top-level OpenAPI security requirements.
func (s *Server) SetDefaultSecurity(requirements OpenAPISecurityRequirements) {
	if s == nil {
		return
	}
	if !s.allowConfigMutation("SetDefaultSecurity") {
		return
	}
	expanded := expandSecurityRequirements(requirements)
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		doc.Security = cloneBuiltInSecurityRequirements(expanded)
	})
}

// RegisterComponentParameter registers a reusable OpenAPI parameter component.
func (s *Server) RegisterComponentParameter(name string, param *huma.Param) {
	if s == nil || name == "" || param == nil {
		return
	}
	if !s.allowConfigMutation("RegisterComponentParameter") {
		return
	}
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		components := ensureComponents(doc)
		if components.Parameters == nil {
			components.Parameters = map[string]*huma.Param{}
		}
		components.Parameters[name] = cloneParam(param)
	})
}

// RegisterComponentHeader registers a reusable OpenAPI header component.
func (s *Server) RegisterComponentHeader(name string, header *huma.Param) {
	if s == nil || name == "" || header == nil {
		return
	}
	if !s.allowConfigMutation("RegisterComponentHeader") {
		return
	}
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		components := ensureComponents(doc)
		if components.Headers == nil {
			components.Headers = map[string]*huma.Header{}
		}
		components.Headers[name] = cloneParam(header)
	})
}

// RegisterGlobalParameter adds a parameter to all current and future operations.
func (s *Server) RegisterGlobalParameter(param *huma.Param) {
	if s == nil || param == nil || param.Name == "" || param.In == "" {
		return
	}
	if !s.allowConfigMutation("RegisterGlobalParameter") {
		return
	}

	cloned := cloneParam(param)
	s.UseOperationModifier(func(op *huma.Operation) {
		appendOperationParameter(op, cloned)
	})
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		forEachOperation(doc, func(op *huma.Operation) {
			appendOperationParameter(op, cloned)
		})
	})
}

// RegisterGlobalHeader adds a request header parameter to all current and future operations.
func (s *Server) RegisterGlobalHeader(header *huma.Param) {
	if header == nil {
		return
	}
	if !s.allowConfigMutation("RegisterGlobalHeader") {
		return
	}
	cloned := cloneParam(header)
	cloned.In = "header"
	s.RegisterGlobalParameter(cloned)
}
