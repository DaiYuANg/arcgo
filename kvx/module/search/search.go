// Package search provides RediSearch functionality.
package search

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
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
	return i.client.CreateIndex(ctx, i.name, i.keyPrefix, i.schema)
}

// Drop drops the search index.
func (i *Index) Drop(ctx context.Context) error {
	return i.client.DropIndex(ctx, i.name)
}

// Search performs a search query on this index.
func (i *Index) Search(ctx context.Context, query string, opts *SearchOptions) (*SearchResult, error) {
	opts = resolveSearchOptions(opts)

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
		return nil, err
	}

	return &SearchResult{
		Keys:  keys,
		Total: int64(len(keys)),
	}, nil
}

// SearchOptions contains options for search queries.
type SearchOptions struct {
	Limit     int
	SortBy    string
	Ascending bool
}

// DefaultSearchOptions returns default search options.
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		Limit:     10,
		Ascending: true,
	}
}

// SearchResult represents the result of a search query.
type SearchResult struct {
	Keys  []string
	Total int64
}

// QueryBuilder helps build search queries.
type QueryBuilder struct {
	parts []string
}

// NewQueryBuilder creates a new QueryBuilder.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		parts: make([]string, 0),
	}
}

// Text adds a text search condition.
func (qb *QueryBuilder) Text(field, value string) *QueryBuilder {
	if field == "" {
		// Full-text search across all fields
		qb.parts = append(qb.parts, value)
	} else {
		qb.parts = append(qb.parts, fmt.Sprintf("@%s:%s", field, value))
	}
	return qb
}

// Tag adds a tag search condition (exact match).
func (qb *QueryBuilder) Tag(field, value string) *QueryBuilder {
	qb.parts = append(qb.parts, fmt.Sprintf("@%s:{%s}", field, escapeTag(value)))
	return qb
}

// Tags adds a tag search condition with multiple values (OR).
func (qb *QueryBuilder) Tags(field string, values []string) *QueryBuilder {
	escaped := lo.Map(values, func(value string, _ int) string {
		return escapeTag(value)
	})
	qb.parts = append(qb.parts, fmt.Sprintf("@%s:{%s}", field, strings.Join(escaped, "|")))
	return qb
}

func resolveSearchOptions(opts *SearchOptions) *SearchOptions {
	return mo.TupleToOption(opts, opts != nil).OrElse(DefaultSearchOptions())
}

// Range adds a numeric range condition.
func (qb *QueryBuilder) Range(field string, min, max float64) *QueryBuilder {
	qb.parts = append(qb.parts, fmt.Sprintf("@%s:[%v %v]", field, formatNumber(min), formatNumber(max)))
	return qb
}

// GreaterThan adds a greater than condition.
func (qb *QueryBuilder) GreaterThan(field string, value float64) *QueryBuilder {
	qb.parts = append(qb.parts, fmt.Sprintf("@%s:[(%v +inf]", field, formatNumber(value)))
	return qb
}

// LessThan adds a less than condition.
func (qb *QueryBuilder) LessThan(field string, value float64) *QueryBuilder {
	qb.parts = append(qb.parts, fmt.Sprintf("@%s:[-inf (%v]", field, formatNumber(value)))
	return qb
}

// And combines conditions with AND.
func (qb *QueryBuilder) And() *QueryBuilder {
	if len(qb.parts) > 1 {
		lastTwo := qb.parts[len(qb.parts)-2:]
		qb.parts = qb.parts[:len(qb.parts)-2]
		qb.parts = append(qb.parts, fmt.Sprintf("(%s) (%s)", lastTwo[0], lastTwo[1]))
	}
	return qb
}

// Or combines conditions with OR.
func (qb *QueryBuilder) Or() *QueryBuilder {
	if len(qb.parts) > 1 {
		lastTwo := qb.parts[len(qb.parts)-2:]
		qb.parts = qb.parts[:len(qb.parts)-2]
		qb.parts = append(qb.parts, fmt.Sprintf("(%s)|(%s)", lastTwo[0], lastTwo[1]))
	}
	return qb
}

// Not negates the last condition.
func (qb *QueryBuilder) Not() *QueryBuilder {
	if len(qb.parts) > 0 {
		last := qb.parts[len(qb.parts)-1]
		qb.parts[len(qb.parts)-1] = fmt.Sprintf("-(%s)", last)
	}
	return qb
}

// Build builds the query string.
func (qb *QueryBuilder) Build() string {
	if len(qb.parts) == 0 {
		return "*"
	}
	if len(qb.parts) == 1 {
		return qb.parts[0]
	}
	// Join multiple parts with AND by default
	return strings.Join(qb.parts, " ")
}

// escapeTag escapes special characters in tag values.
func escapeTag(value string) string {
	// Escape special characters: , . < > { } [ ] " ' : ; ! @ ~ $ % ^ & * ( ) - + = ~ |
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, ",", "\\,")
	value = strings.ReplaceAll(value, ".", "\\.")
	value = strings.ReplaceAll(value, "<", "\\<")
	value = strings.ReplaceAll(value, ">", "\\>")
	value = strings.ReplaceAll(value, "{", "\\{")
	value = strings.ReplaceAll(value, "}", "\\}")
	value = strings.ReplaceAll(value, "[", "\\[")
	value = strings.ReplaceAll(value, "]", "\\]")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "'", "\\'")
	value = strings.ReplaceAll(value, ":", "\\:")
	value = strings.ReplaceAll(value, ";", "\\;")
	value = strings.ReplaceAll(value, "!", "\\!")
	value = strings.ReplaceAll(value, "@", "\\@")
	value = strings.ReplaceAll(value, "~", "\\~")
	value = strings.ReplaceAll(value, "$", "\\$")
	value = strings.ReplaceAll(value, "%", "\\%")
	value = strings.ReplaceAll(value, "^", "\\^")
	value = strings.ReplaceAll(value, "&", "\\&")
	value = strings.ReplaceAll(value, "*", "\\*")
	value = strings.ReplaceAll(value, "(", "\\(")
	value = strings.ReplaceAll(value, ")", "\\)")
	value = strings.ReplaceAll(value, "-", "\\-")
	value = strings.ReplaceAll(value, "+", "\\+")
	value = strings.ReplaceAll(value, "=", "\\=")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}

func formatNumber(n float64) string {
	if n == float64(int64(n)) {
		return strconv.FormatInt(int64(n), 10)
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}

// SchemaBuilder helps build search index schemas.
type SchemaBuilder struct {
	fields []kvx.SchemaField
}

// NewSchemaBuilder creates a new SchemaBuilder.
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		fields: make([]kvx.SchemaField, 0),
	}
}

// TextField adds a text field to the schema.
func (sb *SchemaBuilder) TextField(name string, sortable bool) *SchemaBuilder {
	sb.fields = append(sb.fields, kvx.SchemaField{
		Name:     name,
		Type:     kvx.SchemaFieldTypeText,
		Indexing: true,
		Sortable: sortable,
	})
	return sb
}

// TagField adds a tag field to the schema.
func (sb *SchemaBuilder) TagField(name string, sortable bool) *SchemaBuilder {
	sb.fields = append(sb.fields, kvx.SchemaField{
		Name:     name,
		Type:     kvx.SchemaFieldTypeTag,
		Indexing: true,
		Sortable: sortable,
	})
	return sb
}

// NumericField adds a numeric field to the schema.
func (sb *SchemaBuilder) NumericField(name string, sortable bool) *SchemaBuilder {
	sb.fields = append(sb.fields, kvx.SchemaField{
		Name:     name,
		Type:     kvx.SchemaFieldTypeNumeric,
		Indexing: true,
		Sortable: sortable,
	})
	return sb
}

// Build builds the schema.
func (sb *SchemaBuilder) Build() []kvx.SchemaField {
	return sb.fields
}

// SearchableRepository provides search capabilities for repositories.
type SearchableRepository[T any] struct {
	index     *Index
	keyPrefix string
}

// NewSearchableRepository creates a new SearchableRepository.
func NewSearchableRepository[T any](client kvx.Search, indexName, keyPrefix string, schema []kvx.SchemaField) *SearchableRepository[T] {
	return &SearchableRepository[T]{
		index:     NewIndex(client, indexName, keyPrefix, schema),
		keyPrefix: keyPrefix,
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
func (r *SearchableRepository[T]) Search(ctx context.Context, query string, opts *SearchOptions) (*SearchResult, error) {
	return r.index.Search(ctx, query, opts)
}

// GetIndex returns the underlying index.
func (r *SearchableRepository[T]) GetIndex() *Index {
	return r.index
}
