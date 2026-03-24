package repository

import "github.com/DaiYuANg/arcgo/dbx"

type Base[E any, S dbx.SchemaSource[E]] struct {
	db                  *dbx.DB
	session             dbx.Session
	schema              S
	mapper              dbx.Mapper[E]
	byIDNotFoundAsError bool
}

func (r *Base[E, S]) DB() *dbx.DB { return r.db }
func (r *Base[E, S]) Schema() S   { return r.schema }
func (r *Base[E, S]) Mapper() dbx.Mapper[E] {
	return r.mapper
}

type PageResult[E any] struct {
	Items    []E
	Total    int64
	Page     int
	PageSize int
}

