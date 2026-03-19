package sqltmplx

import (
	"testing"
	"testing/fstest"

	"github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
)

func TestTemplateBindReturnsDBXBoundQuery(t *testing.T) {
	engine := New(sqlite.Dialect{})
	template, err := engine.CompileNamed("user/find_active.sql", `
select id, username
from users
where status = /* status */1
`)
	if err != nil {
		t.Fatalf("CompileNamed returned error: %v", err)
	}

	bound, err := template.Bind(struct {
		Status int
	}{Status: 1})
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}
	if bound.Name != "user/find_active.sql" {
		t.Fatalf("unexpected bound name: %q", bound.Name)
	}
	if bound.SQL == "" || len(bound.Args) != 1 || bound.Args[0] != 1 {
		t.Fatalf("unexpected bound query: %+v", bound)
	}
}

func TestRegistryLoadsAndCachesTemplates(t *testing.T) {
	registry := NewRegistry(fstest.MapFS{
		"sql/user/find_active.sql": {
			Data: []byte(`
select id, username
from users
where status = /* status */1
order by id
`),
		},
	}, sqlite.Dialect{})

	first, err := registry.Template("sql/user/find_active.sql")
	if err != nil {
		t.Fatalf("Template returned error: %v", err)
	}
	second, err := registry.Statement("/sql/user/find_active.sql")
	if err != nil {
		t.Fatalf("Statement returned error: %v", err)
	}
	if first != second {
		t.Fatal("expected registry to reuse cached template instance")
	}

	bound, err := second.Bind(struct {
		Status int
	}{Status: 2})
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}
	if bound.Name != "sql/user/find_active.sql" {
		t.Fatalf("unexpected statement name: %q", bound.Name)
	}
	if len(bound.Args) != 1 || bound.Args[0] != 2 {
		t.Fatalf("unexpected statement args: %#v", bound.Args)
	}
}
