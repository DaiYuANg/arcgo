package redis

import (
	"context"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

// CreateIndex creates a secondary index.
func (a *Adapter) CreateIndex(ctx context.Context, indexName, prefix string, schema []kvx.SchemaField) error {
	args := collectionx.NewListWithCapacity[any](len(schema)*3+8,
		"FT.CREATE", indexName, "ON", "HASH", "PREFIX", 1, prefix, "SCHEMA",
	)
	args.Add(lo.FlatMap(schema, func(field kvx.SchemaField, _ int) []any {
		parts := []any{field.Name, string(field.Type)}
		if field.Sortable {
			parts = lo.Concat(parts, []any{"SORTABLE"})
		}
		return parts
	})...)

	return wrapRedisError("create search index", a.client.Do(ctx, args.Values()...).Err())
}

// DropIndex drops a secondary index.
func (a *Adapter) DropIndex(ctx context.Context, indexName string) error {
	return wrapRedisError("drop search index", a.client.Do(ctx, "FT.DROPINDEX", indexName).Err())
}

// Search performs a search query.
func (a *Adapter) Search(ctx context.Context, indexName, query string, limit int) ([]string, error) {
	val, err := a.client.Do(ctx, "FT.SEARCH", indexName, query, "LIMIT", 0, limit).Result()
	val, err = wrapRedisResult("search index", val, err)
	if err != nil {
		return nil, err
	}

	return parseFTSearchResponse(val), nil
}

// SearchWithSort performs a search query with sorting.
func (a *Adapter) SearchWithSort(ctx context.Context, indexName, query, sortBy string, ascending bool, limit int) ([]string, error) {
	args := collectionx.NewList[any]("FT.SEARCH", indexName, query, "SORTBY", sortBy)
	if !ascending {
		args.Add("DESC")
	}
	args.Add("LIMIT", 0, limit)

	val, err := a.client.Do(ctx, args.Values()...).Result()
	val, err = wrapRedisResult("search index with sort", val, err)
	if err != nil {
		return nil, err
	}

	return parseFTSearchResponse(val), nil
}

// SearchAggregate performs an aggregation query.
func (a *Adapter) SearchAggregate(ctx context.Context, indexName, query string, limit int) ([]map[string]any, error) {
	val, err := a.client.Do(ctx, "FT.AGGREGATE", indexName, query, "LIMIT", 0, limit).Result()
	val, err = wrapRedisResult("aggregate search index", val, err)
	if err != nil {
		return nil, err
	}

	return parseFTAggregateResponse(val), nil
}
