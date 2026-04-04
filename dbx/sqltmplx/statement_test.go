package sqltmplx_test

import (
	"testing"
	"testing/fstest"

	"github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
	sqltmplx "github.com/DaiYuANg/arcgo/dbx/sqltmplx"
	"github.com/stretchr/testify/require"
)

func TestTemplateBindReturnsDBXBoundQuery(t *testing.T) {
	engine := sqltmplx.New(sqlite.New())
	template, err := engine.CompileNamed("user/find_active.sql", `
select id, username
from users
where status = /* status */1
`)
	require.NoError(t, err)

	bound, err := template.Bind(struct {
		Status int
	}{Status: 1})
	require.NoError(t, err)
	require.Equal(t, "user/find_active.sql", bound.Name)
	require.Equal(t, 1, bound.Args.Len())
	value, ok := bound.Args.Get(0)
	require.True(t, ok)
	require.Equal(t, 1, value)
	require.NotEmpty(t, bound.SQL)
}

func TestRegistryLoadsAndCachesTemplates(t *testing.T) {
	registry := sqltmplx.NewRegistry(fstest.MapFS{
		"sql/user/find_active.sql": {
			Data: []byte(`
select id, username
from users
where status = /* status */1
order by id
`),
		},
	}, sqlite.New())

	first, err := registry.Template("sql/user/find_active.sql")
	require.NoError(t, err)

	second, err := registry.Statement("/sql/user/find_active.sql")
	require.NoError(t, err)
	require.Same(t, first, second)

	bound, err := second.Bind(struct {
		Status int
	}{Status: 2})
	require.NoError(t, err)
	require.Equal(t, "sql/user/find_active.sql", bound.Name)
	require.Equal(t, []any{2}, bound.Args.Values())
}
