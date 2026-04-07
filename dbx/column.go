package dbx

import (
	"reflect"
	"strings"

	"github.com/samber/lo"
)

type Ref[E any, T any] interface {
	Name() string
	refNode()
}

type ReferentialAction string

type IDStrategy string

type IDMarker interface {
	idStrategy() IDStrategy
	uuidVersion() string
}

const (
	ReferentialNoAction   ReferentialAction = "NO ACTION"
	ReferentialRestrict   ReferentialAction = "RESTRICT"
	ReferentialCascade    ReferentialAction = "CASCADE"
	ReferentialSetNull    ReferentialAction = "SET NULL"
	ReferentialSetDefault ReferentialAction = "SET DEFAULT"

	IDStrategyUnset     IDStrategy = ""
	IDStrategyDBAuto    IDStrategy = "db_auto"
	IDStrategySnowflake IDStrategy = "snowflake"
	IDStrategyUUID      IDStrategy = "uuid"
	IDStrategyULID      IDStrategy = "ulid"
	IDStrategyKSUID     IDStrategy = "ksuid"
	DefaultUUIDVersion             = "v7"
)

type ForeignKeyRef struct {
	TargetTable  string
	TargetColumn string
	OnDelete     ReferentialAction
	OnUpdate     ReferentialAction
}

type ColumnMeta struct {
	Name          string
	Table         string
	Alias         string
	FieldName     string
	GoType        reflect.Type
	SQLType       string
	PrimaryKey    bool
	AutoIncrement bool
	Nullable      bool
	Unique        bool
	Indexed       bool
	DefaultValue  string
	References    *ForeignKeyRef
	IDStrategy    IDStrategy
	UUIDVersion   string
}

func cloneColumnMeta(meta ColumnMeta) ColumnMeta {
	if meta.References == nil {
		return meta
	}
	meta.References = new(*meta.References)
	return meta
}

type columnBinder interface {
	bindColumn(binding columnBinding) any
}

type columnAccessor interface {
	columnRef() ColumnMeta
}

type columnTypeReporter interface {
	valueType() reflect.Type
}

type typedColumn[T any] interface {
	columnRef() ColumnMeta
}

type columnBinding struct {
	meta ColumnMeta
}

type Column[E any, T any] struct {
	meta ColumnMeta
}

// IDColumn declares an ID policy directly in the schema field type.
// The marker strategy is applied during schema binding.
type IDColumn[E any, T any, M IDMarker] struct {
	Column[E, T]
}

type ColumnOption[E any, T any] func(Column[E, T]) Column[E, T]

type IDAuto struct{}
type IDSnowflake struct{}
type IDUUID struct{}
type IDUUIDv7 struct{}
type IDUUIDv4 struct{}
type IDULID struct{}
type IDKSUID struct{}

func (IDAuto) idStrategy() IDStrategy      { return IDStrategyDBAuto }
func (IDAuto) uuidVersion() string         { return "" }
func (IDSnowflake) idStrategy() IDStrategy { return IDStrategySnowflake }
func (IDSnowflake) uuidVersion() string    { return "" }
func (IDUUID) idStrategy() IDStrategy      { return IDStrategyUUID }
func (IDUUID) uuidVersion() string         { return "" }
func (IDUUIDv7) idStrategy() IDStrategy    { return IDStrategyUUID }
func (IDUUIDv7) uuidVersion() string       { return "v7" }
func (IDUUIDv4) idStrategy() IDStrategy    { return IDStrategyUUID }
func (IDUUIDv4) uuidVersion() string       { return "v4" }
func (IDULID) idStrategy() IDStrategy      { return IDStrategyULID }
func (IDULID) uuidVersion() string         { return "" }
func (IDKSUID) idStrategy() IDStrategy     { return IDStrategyKSUID }
func (IDKSUID) uuidVersion() string        { return "" }

func NewColumn[E any, T any](opts ...ColumnOption[E, T]) Column[E, T] {
	column := Column[E, T]{}
	lo.ForEach(lo.Filter(opts, func(opt ColumnOption[E, T], _ int) bool { return opt != nil }), func(opt ColumnOption[E, T], _ int) {
		column = opt(column)
	})
	return column
}

func (c IDColumn[E, T, M]) bindColumn(binding columnBinding) any {
	marker := *new(M)
	base := c.Column
	base.meta.PrimaryKey = true
	base.meta.IDStrategy = marker.idStrategy()
	if version := marker.uuidVersion(); version != "" {
		base.meta.UUIDVersion = version
	}
	boundValue := base.bindColumn(binding)
	bound, ok := boundValue.(Column[E, T])
	if !ok {
		return IDColumn[E, T, M]{Column: base}
	}
	return IDColumn[E, T, M]{Column: bound}
}

func NamedColumn[T any](source TableSource, name string) Column[struct{}, T] {
	table := source.tableRef()
	return Column[struct{}, T]{
		meta: ColumnMeta{
			Name:   strings.TrimSpace(name),
			Table:  table.Name(),
			Alias:  table.Alias(),
			GoType: reflect.TypeFor[T](),
		},
	}
}

func ResultColumn[T any](name string) Column[struct{}, T] {
	return Column[struct{}, T]{
		meta: ColumnMeta{
			Name:   strings.TrimSpace(name),
			GoType: reflect.TypeFor[T](),
		},
	}
}
