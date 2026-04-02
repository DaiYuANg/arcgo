package repository

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
)

// EntitySchema is the schema contract required by repository generic code.
type EntitySchema[E any] interface {
	dbx.SchemaSource[E]
	TableName() string
}

// Base provides generic CRUD helpers for a schema-backed entity type.
type Base[E any, S EntitySchema[E]] struct {
	db                  *dbx.DB
	session             dbx.Session
	schema              S
	mapper              dbx.Mapper[E]
	byIDNotFoundAsError bool
}

// DB returns the database bound to the repository.
func (r *Base[E, S]) DB() *dbx.DB { return r.db }

// Schema returns the schema bound to the repository.
func (r *Base[E, S]) Schema() S { return r.schema }

// Mapper returns the mapper used to persist and scan entities.
func (r *Base[E, S]) Mapper() dbx.Mapper[E] {
	return r.mapper
}

// PageResult contains the items and metadata for a paginated query.
type PageResult[E any] struct {
	Items    collectionx.List[E]
	Total    int64
	Page     int
	PageSize int
}
