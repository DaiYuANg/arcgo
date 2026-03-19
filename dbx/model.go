package dbx

import (
	"database/sql"
	"reflect"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

type RowsScanner[E any] interface {
	ScanRows(rows *sql.Rows) ([]E, error)
}

type StructMapper[E any] struct {
	meta *mapperMetadata
}

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
	fields := collectionx.NewListWithCapacity[MappedField](len(mappedFields))
	byColumn := collectionx.NewMapWithCapacity[string, MappedField](len(mappedFields))
	lo.ForEach(mappedFields, func(field MappedField, _ int) {
		fields.Add(field)
		byColumn.Set(field.Column, field)
	})

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
