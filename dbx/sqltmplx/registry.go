package sqltmplx

import (
	"io/fs"
	"path"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type Registry struct {
	engine *Engine
	fsys   fs.FS
	cache  collectionx.ConcurrentMap[string, *Template]
}

func NewRegistry(fsys fs.FS, d dialect.Contract, opts ...Option) *Registry {
	return &Registry{
		engine: New(d, opts...),
		fsys:   fsys,
		cache:  collectionx.NewConcurrentMap[string, *Template](),
	}
}

func (r *Registry) Template(name string) (*Template, error) {
	if r == nil || r.engine == nil || r.fsys == nil {
		return nil, dbx.ErrNilStatement
	}

	normalized := normalizeTemplateName(name)
	if cached, ok := r.cache.Get(normalized); ok {
		return cached, nil
	}

	content, err := fs.ReadFile(r.fsys, normalized)
	if err != nil {
		return nil, err
	}
	template, err := r.engine.CompileNamed(normalized, string(content))
	if err != nil {
		return nil, err
	}

	actual, _ := r.cache.GetOrStore(normalized, template)
	return actual, nil
}

func (r *Registry) MustTemplate(name string) *Template {
	template, err := r.Template(name)
	if err != nil {
		panic(err)
	}
	return template
}

func (r *Registry) Statement(name string) (*Template, error) {
	return r.Template(name)
}

func (r *Registry) MustStatement(name string) *Template {
	return r.MustTemplate(name)
}

func normalizeTemplateName(name string) string {
	normalized := path.Clean(strings.TrimSpace(name))
	return strings.TrimPrefix(normalized, "/")
}
