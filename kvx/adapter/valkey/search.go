package valkey

import (
	"context"
	"strconv"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

// CreateIndex creates a secondary index.
func (a *Adapter) CreateIndex(ctx context.Context, indexName, prefix string, schema []kvx.SchemaField) error {
	args := collectionx.NewListWithCapacity[string](len(schema)*3+7, indexName, "ON", "HASH", "PREFIX", "1", prefix, "SCHEMA")
	args.Add(lo.FlatMap(schema, func(field kvx.SchemaField, _ int) []string {
		parts := []string{field.Name, string(field.Type)}
		if field.Sortable {
			parts = lo.Concat(parts, []string{"SORTABLE"})
		}
		return parts
	})...)

	return wrapValkeyError("create search index", a.client.Do(ctx, a.client.B().Arbitrary("FT.CREATE").Args(args.Values()...).Build()).Error())
}

// DropIndex drops a secondary index.
func (a *Adapter) DropIndex(ctx context.Context, indexName string) error {
	return wrapValkeyError("drop search index", a.client.Do(ctx, a.client.B().Arbitrary("FT.DROPINDEX").Args(indexName).Build()).Error())
}

// Search performs a search query.
func (a *Adapter) Search(ctx context.Context, indexName, query string, limit int) (collectionx.List[string], error) {
	resp := a.client.Do(ctx, a.client.B().Arbitrary("FT.SEARCH").Args(indexName, query, "LIMIT", "0", strconv.Itoa(limit)).Build())
	docs, err := ftSearchDocsFromResult("search index", resp)
	if err != nil {
		return nil, err
	}

	return searchDocsToKeys(docs), nil
}

// SearchWithSort performs a search query with sorting.
func (a *Adapter) SearchWithSort(ctx context.Context, indexName, query, sortBy string, ascending bool, limit int) (collectionx.List[string], error) {
	args := collectionx.NewList[string](indexName, query, "SORTBY", sortBy)
	if !ascending {
		args.Add("DESC")
	}
	args.Add("LIMIT", "0", strconv.Itoa(limit))

	resp := a.client.Do(ctx, a.client.B().Arbitrary("FT.SEARCH").Args(args.Values()...).Build())
	docs, err := ftSearchDocsFromResult("search index with sort", resp)
	if err != nil {
		return nil, err
	}

	return searchDocsToKeys(docs), nil
}

// SearchAggregate performs an aggregation query.
func (a *Adapter) SearchAggregate(ctx context.Context, indexName, query string, limit int) ([]map[string]any, error) {
	resp := a.client.Do(ctx, a.client.B().Arbitrary("FT.AGGREGATE").Args(indexName, query, "LIMIT", "0", strconv.Itoa(limit)).Build())
	docs, err := ftAggregateDocsFromResult("aggregate search index", resp)
	if err != nil {
		return nil, err
	}

	return aggregateDocsToRows(docs), nil
}
