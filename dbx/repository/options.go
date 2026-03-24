package repository

import "github.com/DaiYuANg/arcgo/dbx"

type Option func(*baseOptions)

type baseOptions struct {
	byIDNotFoundAsError bool
}

func defaultOptions() baseOptions { return baseOptions{} }

func WithByIDNotFoundAsError(enabled bool) Option {
	return func(opts *baseOptions) { opts.byIDNotFoundAsError = enabled }
}

func New[E any, S dbx.SchemaSource[E]](db *dbx.DB, schema S) *Base[E, S] {
	return NewWithOptions[E](db, schema)
}

func NewWithOptions[E any, S dbx.SchemaSource[E]](db *dbx.DB, schema S, opts ...Option) *Base[E, S] {
	config := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	return &Base[E, S]{
		db:                  db,
		session:             db,
		schema:              schema,
		mapper:              dbx.MustMapper[E](schema),
		byIDNotFoundAsError: config.byIDNotFoundAsError,
	}
}

