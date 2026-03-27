package redis

import (
	"context"

	"github.com/DaiYuANg/arcgo/kvx"
)

// CreateIndex creates a secondary index.
func (a *Adapter) CreateIndex(ctx context.Context, indexName, prefix string, schema []kvx.SchemaField) error {
	args := make([]any, 0, len(schema)*3+8)
	args = append(args, "FT.CREATE", indexName, "ON", "HASH", "PREFIX", 1, prefix, "SCHEMA")

	for _, f := range schema {
		args = append(args, f.Name, string(f.Type))
		if f.Sortable {
			args = append(args, "SORTABLE")
		}
	}

	return wrapRedisError("create search index", a.client.Do(ctx, args...).Err())
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
	args := []any{"FT.SEARCH", indexName, query, "SORTBY", sortBy}
	if !ascending {
		args = append(args, "DESC")
	}
	args = append(args, "LIMIT", 0, limit)

	val, err := a.client.Do(ctx, args...).Result()
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
