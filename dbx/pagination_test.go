package dbx_test

import (
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
	dbx "github.com/DaiYuANg/arcgo/dbx"
	"github.com/stretchr/testify/require"
)

func TestPageRequestNormalizeAndApply(t *testing.T) {
	request := dbx.Page(0, 0).WithMaxPageSize(10)
	require.Equal(t, 1, request.Page)
	require.Equal(t, 10, request.PageSize)
	require.Equal(t, 0, request.Offset())

	users := MustSchema("users", UserSchema{})
	query := Select(users.ID, users.Username).
		From(users).
		OrderBy(users.ID.Asc()).
		PageBy(2, 5)

	bound, err := query.Build(testSQLiteDialect{})
	require.NoError(t, err)
	require.Equal(t, `SELECT "users"."id", "users"."username" FROM "users" ORDER BY "users"."id" ASC LIMIT 5 OFFSET 5`, bound.SQL)
}

func TestNewPageResultBuildsMetadata(t *testing.T) {
	result := dbx.NewPageResult(collectionx.NewList("alice"), 21, dbx.Page(2, 10))

	require.EqualValues(t, 21, result.Total)
	require.Equal(t, 1, result.Items.Len())
	require.Equal(t, 2, result.Page)
	require.Equal(t, 10, result.PageSize)
	require.Equal(t, 10, result.Offset)
	require.Equal(t, 3, result.TotalPages)
	require.True(t, result.HasNext)
	require.True(t, result.HasPrevious)
}
