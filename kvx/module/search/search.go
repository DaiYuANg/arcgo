// Package search provides RediSearch functionality.
package search

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/mo"
)

// Search provides high-level search operations.
type Search struct {
	client kvx.Search
}

// NewSearch creates a new Search instance.
func NewSearch(client kvx.Search) *Search {
	return &Search{client: client}
}

// Index represents a search index.
type Index struct {
	client    kvx.Search
	name      string
	keyPrefix string
	schema    []kvx.SchemaField
}

// NewIndex creates a new Index instance.
func NewIndex(client kvx.Search, name, keyPrefix string, schema []kvx.SchemaField) *Index {
	return &Index{
		client:    client,
		name:      name,
		keyPrefix: keyPrefix,
		schema:    schema,
	}
}

// Create creates the search index.
func (i *Index) Create(ctx context.Context) error {
	if err := i.client.CreateIndex(ctx, i.name, i.keyPrefix, i.schema); err != nil {
		return fmt.Errorf("create search index %q: %w", i.name, err)
	}
	return nil
}

// Drop drops the search index.
func (i *Index) Drop(ctx context.Context) error {
	if err := i.client.DropIndex(ctx, i.name); err != nil {
		return fmt.Errorf("drop search index %q: %w", i.name, err)
	}
	return nil
}

// Search performs a search query on this index.
func (i *Index) Search(ctx context.Context, query string, opts *Options) (*Result, error) {
	opts = resolveOptions(opts)

	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	var keys []string
	var err error

	if opts.SortBy != "" {
		keys, err = i.client.SearchWithSort(ctx, i.name, query, opts.SortBy, opts.Ascending, limit)
	} else {
		keys, err = i.client.Search(ctx, i.name, query, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("search index %q with query %q: %w", i.name, query, err)
	}

	return &Result{
		Keys:  keys,
		Total: int64(len(keys)),
	}, nil
}

// Options contains options for search queries.
type Options struct {
	Limit     int
	SortBy    string
	Ascending bool
}

// DefaultOptions returns default search options.
func DefaultOptions() *Options {
	return &Options{
		Limit:     10,
		Ascending: true,
	}
}

// Result represents the result of a search query.
type Result struct {
	Keys  []string
	Total int64
}

func resolveOptions(opts *Options) *Options {
	return mo.TupleToOption(opts, opts != nil).OrElse(DefaultOptions())
}
