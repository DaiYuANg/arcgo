package dbx

import (
	"errors"
	"fmt"
)

var (
	ErrNilDB           = errors.New("dbx: db is nil")
	ErrNilSQLDB        = errors.New("dbx: sql.DB is nil")
	ErrMissingDriver   = errors.New("dbx: Open requires WithDriver")
	ErrMissingDSN      = errors.New("dbx: Open requires WithDSN")
	ErrMissingDialect  = errors.New("dbx: Open requires WithDialect")
	ErrIDGeneratorNodeIDConflict = errors.New("dbx: WithIDGenerator and WithNodeID cannot be used together")
	ErrInvalidNodeID = errors.New("dbx: node id is out of range")
	ErrNilDialect   = errors.New("dbx: dialect is nil")
	ErrNilCodec     = errors.New("dbx: codec is nil")
	ErrNilMapper    = errors.New("dbx: mapper is nil")
	ErrNilStatement = errors.New("dbx: sql statement is nil")
	ErrNilEntity    = errors.New("dbx: entity is nil")
	ErrTooManyRows  = errors.New("dbx: query returned more than one row")
	ErrNoPrimaryKey = errors.New("dbx: schema does not define a primary key")
	ErrUnknownCodec = errors.New("dbx: codec is not registered")
	ErrUnmappedColumn = errors.New("dbx: result column is not mapped to entity")
	ErrPrimaryKeyUnmapped = errors.New("dbx: primary key column is not mapped to entity")
	ErrUnsupportedEntity = errors.New("dbx: entity type must be a struct")
	ErrUnsupportedSchema = errors.New("dbx: schema type is unsupported")
)

// PrimaryKeyUnmappedError carries the column name when a primary key column
// is not mapped to the entity. Use errors.As to extract the column for programmatic handling.
type PrimaryKeyUnmappedError struct {
	Column string
}

func (e *PrimaryKeyUnmappedError) Error() string {
	if e.Column != "" {
		return fmt.Sprintf("dbx: primary key column %q is not mapped to entity", e.Column)
	}
	return "dbx: primary key column is not mapped to entity"
}

func (e *PrimaryKeyUnmappedError) Unwrap() error {
	return ErrPrimaryKeyUnmapped
}

// UnknownCodecError carries the codec name when a codec is not registered.
// Use errors.As to extract the name for programmatic handling.
type UnknownCodecError struct {
	Name string
}

func (e *UnknownCodecError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("dbx: codec %q is not registered", e.Name)
	}
	return "dbx: codec is not registered"
}

func (e *UnknownCodecError) Unwrap() error {
	return ErrUnknownCodec
}

// UnmappedColumnError carries the column name when a result column is not
// mapped to the entity. Use errors.As to extract the column for programmatic handling.
type UnmappedColumnError struct {
	Column string
}

func (e *UnmappedColumnError) Error() string {
	if e.Column != "" {
		return fmt.Sprintf("dbx: result column %q is not mapped to entity", e.Column)
	}
	return "dbx: result column is not mapped to entity"
}

func (e *UnmappedColumnError) Unwrap() error {
	return ErrUnmappedColumn
}

// NodeIDOutOfRangeError carries the out-of-range node id and supported range.
// Use errors.Is(err, ErrInvalidNodeID) or errors.As(err, *NodeIDOutOfRangeError).
type NodeIDOutOfRangeError struct {
	NodeID uint16
	Min    uint16
	Max    uint16
}

func (e *NodeIDOutOfRangeError) Error() string {
	return fmt.Sprintf("dbx: node id %d out of range [%d,%d]", e.NodeID, e.Min, e.Max)
}

func (e *NodeIDOutOfRangeError) Unwrap() error {
	return ErrInvalidNodeID
}
