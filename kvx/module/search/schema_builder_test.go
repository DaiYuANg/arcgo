package search_test

import (
	"context"
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/module/search"
	"github.com/stretchr/testify/require"
)

func TestSchemaBuilderBuildReturnsCollectionxList(t *testing.T) {
	t.Parallel()

	fields := search.NewSchemaBuilder().
		TextField("title", true).
		TagField("category", false).
		NumericField("score", true).
		Build()

	require.Equal(t, 3, fields.Len())

	first, ok := fields.GetFirst()
	require.True(t, ok)
	require.Equal(t, kvx.SchemaField{
		Name:     "title",
		Type:     kvx.SchemaFieldTypeText,
		Indexing: true,
		Sortable: true,
	}, first)

	last, ok := fields.GetLast()
	require.True(t, ok)
	require.Equal(t, kvx.SchemaField{
		Name:     "score",
		Type:     kvx.SchemaFieldTypeNumeric,
		Indexing: true,
		Sortable: true,
	}, last)
}

func TestIndexCreateConsumesCollectionxListSchema(t *testing.T) {
	t.Parallel()

	client := &stubSearchClient{}
	schema := search.NewSchemaBuilder().
		TextField("title", true).
		TagField("category", false).
		Build()

	index := search.NewIndex(client, "articles", "article:", schema)

	require.NoError(t, index.Create(context.Background()))
	require.Equal(t, []kvx.SchemaField{
		{
			Name:     "title",
			Type:     kvx.SchemaFieldTypeText,
			Indexing: true,
			Sortable: true,
		},
		{
			Name:     "category",
			Type:     kvx.SchemaFieldTypeTag,
			Indexing: true,
			Sortable: false,
		},
	}, client.createdSchema)
}

type stubSearchClient struct {
	createdSchema []kvx.SchemaField
}

func (s *stubSearchClient) CreateIndex(_ context.Context, _, _ string, schema []kvx.SchemaField) error {
	s.createdSchema = append([]kvx.SchemaField(nil), schema...)
	return nil
}

func (*stubSearchClient) DropIndex(context.Context, string) error {
	return nil
}

func (*stubSearchClient) Search(context.Context, string, string, int) (collectionx.List[string], error) {
	return collectionx.NewList[string](), nil
}

func (*stubSearchClient) SearchWithSort(context.Context, string, string, string, bool, int) (collectionx.List[string], error) {
	return collectionx.NewList[string](), nil
}

func (*stubSearchClient) SearchAggregate(context.Context, string, string, int) ([]map[string]any, error) {
	return nil, nil
}
