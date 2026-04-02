package search

import "context"

import "github.com/DaiYuANg/arcgo/collectionx"
import "github.com/DaiYuANg/arcgo/kvx"

// SearchableRepository provides search capabilities for repositories.
type SearchableRepository[T any] struct {
	index *Index
}

// NewSearchableRepository creates a new SearchableRepository.
func NewSearchableRepository[T any](client kvx.Search, indexName, keyPrefix string, schema collectionx.List[kvx.SchemaField]) *SearchableRepository[T] {
	return &SearchableRepository[T]{
		index: NewIndex(client, indexName, keyPrefix, schema),
	}
}

// CreateIndex creates the search index.
func (r *SearchableRepository[T]) CreateIndex(ctx context.Context) error {
	return r.index.Create(ctx)
}

// DropIndex drops the search index.
func (r *SearchableRepository[T]) DropIndex(ctx context.Context) error {
	return r.index.Drop(ctx)
}

// Search searches for entities using the index.
func (r *SearchableRepository[T]) Search(ctx context.Context, query string, opts *Options) (*Result, error) {
	return r.index.Search(ctx, query, opts)
}

// GetIndex returns the underlying index.
func (r *SearchableRepository[T]) GetIndex() *Index {
	return r.index
}
