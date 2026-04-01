package dbx

import (
	"database/sql"
	"reflect"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

// RowsScanner is the schema-less contract for mapping query result rows to entities.
// Used by SQLList, SQLGet, QueryAll, QueryCursor, etc. Both StructMapper and Mapper implement it.
type RowsScanner[E any] interface {
	ScanRows(rows *sql.Rows) ([]E, error)
}

// CapacityHintScanner is an optional extension. When implemented and BoundQuery.CapacityHint > 0,
// QueryAllBound uses ScanRowsWithCapacity to pre-allocate the result slice (reduces append growth).
type CapacityHintScanner[E any] interface {
	ScanRowsWithCapacity(rows *sql.Rows, capacityHint int) ([]E, error)
}

// StructMapper provides schema-less pure DTO mapping. It infers fields from struct tags (e.g. dbx)
// and maps result columns by name. Use for arbitrary SQL when no Schema is available.
// Dependency: StructMapper does not depend on Schema.
type StructMapper[E any] struct {
	meta *mapperMetadata
}

// Mapper extends StructMapper with a schema-derived field subset. It filters StructMapper's fields
// to only those present in the schema columns. Required for CRUD, relation load, and other
// schema-aware operations. Dependency: Mapper depends on Schema (created via NewMapper(schema)).
type Mapper[E any] struct {
	StructMapper[E]
	fields   collectionx.List[MappedField]
	byColumn collectionx.Map[string, MappedField]
}

type MappedField struct {
	Name       string
	Column     string
	Codec      string
	Index      int
	Path       []int
	Type       reflect.Type
	Insertable bool
	Updatable  bool
	codec      Codec
}

// NewStructMapper creates a schema-less mapper for pure DTO mapping (e.g. SQLList, SQLGet with arbitrary SQL).
func NewStructMapper[E any]() (StructMapper[E], error) {
	return NewStructMapperWithOptions[E]()
}

func NewStructMapperWithOptions[E any](opts ...MapperOption) (StructMapper[E], error) {
	config, err := applyMapperOptions(opts...)
	if err != nil {
		return StructMapper[E]{}, err
	}
	meta, err := getOrBuildMapperMetadata[E](config.runtime)
	if err != nil {
		return StructMapper[E]{}, err
	}
	return StructMapper[E]{meta: meta}, nil
}

func MustStructMapper[E any]() StructMapper[E] {
	mapper, err := NewStructMapper[E]()
	if err != nil {
		panic(err)
	}
	return mapper
}

func MustStructMapperWithOptions[E any](opts ...MapperOption) StructMapper[E] {
	mapper, err := NewStructMapperWithOptions[E](opts...)
	if err != nil {
		panic(err)
	}
	return mapper
}

func MustMapper[E any](schema SchemaResource) Mapper[E] {
	mapper, err := NewMapper[E](schema)
	if err != nil {
		panic(err)
	}
	return mapper
}

// NewMapper creates a schema-bound mapper. Use when you have a Schema (CRUD, relation load, repository).
func NewMapper[E any](schema SchemaResource) (Mapper[E], error) {
	return NewMapperWithOptions[E](schema)
}

func MustMapperWithOptions[E any](schema SchemaResource, opts ...MapperOption) Mapper[E] {
	mapper, err := NewMapperWithOptions[E](schema, opts...)
	if err != nil {
		panic(err)
	}
	return mapper
}

func NewMapperWithOptions[E any](schema SchemaResource, opts ...MapperOption) (Mapper[E], error) {
	structMapper, err := NewStructMapperWithOptions[E](opts...)
	if err != nil {
		return Mapper[E]{}, err
	}

	mappedFields := lo.FilterMap(schema.schemaRef().columns, func(column ColumnMeta, _ int) (MappedField, bool) {
		return structMapper.meta.byColumn.Get(column.Name)
	})
	fields := collectionx.NewListWithCapacity[MappedField](len(mappedFields), mappedFields...)
	byColumn := collectionx.NewMapFrom(lo.Associate(mappedFields, func(field MappedField) (string, MappedField) {
		return field.Column, field
	}))

	return Mapper[E]{
		StructMapper: structMapper,
		fields:       fields,
		byColumn:     byColumn,
	}, nil
}

func (m Mapper[E]) Fields() []MappedField {
	if m.byColumn.Len() == 0 {
		return nil
	}
	return m.fields.Values()
}

func (m Mapper[E]) FieldByColumn(column string) (MappedField, bool) {
	if m.byColumn.Len() == 0 {
		return MappedField{}, false
	}
	return m.byColumn.Get(column)
}

func (m StructMapper[E]) Fields() []MappedField {
	if m.meta == nil {
		return nil
	}
	return m.meta.fields.Values()
}

func (m StructMapper[E]) FieldByColumn(column string) (MappedField, bool) {
	if m.meta == nil {
		return MappedField{}, false
	}
	return m.meta.byColumn.Get(column)
}
