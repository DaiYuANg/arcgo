package dbx

import "errors"

var (
	ErrNilDB              = errors.New("dbx: db is nil")
	ErrNilSQLDB           = errors.New("dbx: sql.DB is nil")
	ErrNilDialect         = errors.New("dbx: dialect is nil")
	ErrNilCodec           = errors.New("dbx: codec is nil")
	ErrNilMapper          = errors.New("dbx: mapper is nil")
	ErrNilStatement       = errors.New("dbx: sql statement is nil")
	ErrNilEntity          = errors.New("dbx: entity is nil")
	ErrTooManyRows        = errors.New("dbx: query returned more than one row")
	ErrNoPrimaryKey       = errors.New("dbx: schema does not define a primary key")
	ErrPrimaryKeyUnmapped = errors.New("dbx: primary key column is not mapped to entity")
	ErrUnknownCodec       = errors.New("dbx: codec is not registered")
	ErrUnmappedColumn     = errors.New("dbx: result column is not mapped to entity")
	ErrUnsupportedEntity  = errors.New("dbx: entity type must be a struct")
	ErrUnsupportedSchema  = errors.New("dbx: schema type is unsupported")
)
