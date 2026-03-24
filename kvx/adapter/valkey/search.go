package valkey

import (
	"context"
	"fmt"
	"github.com/DaiYuANg/arcgo/kvx"
)

// ============== Search Interface ==============

// CreateIndex creates a secondary index.
func (a *Adapter) CreateIndex(ctx context.Context, indexName string, prefix string, schema []kvx.SchemaField) error {
	args := []string{indexName, "ON", "HASH", "PREFIX", "1", prefix, "SCHEMA"}

	for _, f := range schema {
		args = append(args, f.Name, string(f.Type))
		if f.Sortable {
			args = append(args, "SORTABLE")
		}
	}

	return a.client.Do(ctx, a.client.B().Arbitrary("FT.CREATE").Args(args...).Build()).Error()
}

// DropIndex drops a secondary index.
func (a *Adapter) DropIndex(ctx context.Context, indexName string) error {
	return a.client.Do(ctx, a.client.B().Arbitrary("FT.DROPINDEX").Args(indexName).Build()).Error()
}

// Search performs a search query.
func (a *Adapter) Search(ctx context.Context, indexName string, query string, limit int) ([]string, error) {
	resp := a.client.Do(ctx, a.client.B().Arbitrary("FT.SEARCH").Args(indexName, query, "LIMIT", "0", fmt.Sprintf("%d", limit)).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	// Use AsFtSearch to parse the response
	total, docs, err := resp.AsFtSearch()
	if err != nil {
		return nil, err
	}
	_ = total // We don't need the total count here

	keys := make([]string, len(docs))
	for i, doc := range docs {
		keys[i] = doc.Key
	}
	return keys, nil
}

// SearchWithSort performs a search query with sorting.
func (a *Adapter) SearchWithSort(ctx context.Context, indexName string, query string, sortBy string, ascending bool, limit int) ([]string, error) {
	args := []string{indexName, query, "SORTBY", sortBy}
	if !ascending {
		args = append(args, "DESC")
	}
	args = append(args, "LIMIT", "0", fmt.Sprintf("%d", limit))

	resp := a.client.Do(ctx, a.client.B().Arbitrary("FT.SEARCH").Args(args...).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	_, docs, err := resp.AsFtSearch()
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(docs))
	for i, doc := range docs {
		keys[i] = doc.Key
	}
	return keys, nil
}

// SearchAggregate performs an aggregation query.
func (a *Adapter) SearchAggregate(ctx context.Context, indexName string, query string, limit int) ([]map[string]interface{}, error) {
	resp := a.client.Do(ctx, a.client.B().Arbitrary("FT.AGGREGATE").Args(indexName, query, "LIMIT", "0", fmt.Sprintf("%d", limit)).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	_, docs, err := resp.AsFtAggregate()
	if err != nil {
		return nil, err
	}
	results := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		row := make(map[string]interface{}, len(doc))
		for k, v := range doc {
			row[k] = v
		}
		results[i] = row
	}
	return results, nil
}
