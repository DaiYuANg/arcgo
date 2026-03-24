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
	bound := base.bindColumn(binding).(Column[E, T])
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

func PrimaryKeyColumn[E any, T any]() ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.PrimaryKey = true
		return column
	}
}

func AutoIncrementColumn[E any, T any]() ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.AutoIncrement = true
		return column
	}
}

func NullableColumn[E any, T any]() ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.Nullable = true
		return column
	}
}

func UniqueColumn[E any, T any]() ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.Unique = true
		return column
	}
}

func IndexedColumn[E any, T any]() ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.Indexed = true
		return column
	}
}

func WithDefault[E any, T any](value string) ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.DefaultValue = value
		return column
	}
}

func WithReference[E any, T any](ref ForeignKeyRef) ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.References = new(ref)
		return column
	}
}

func WithIDStrategyColumn[E any, T any](strategy IDStrategy) ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.IDStrategy = strategy
		return column
	}
}

func WithUUIDVersionColumn[E any, T any](version string) ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.UUIDVersion = strings.TrimSpace(version)
		return column
	}
}

func DBAutoIDColumn[E any, T any]() ColumnOption[E, T] {
	return WithIDStrategyColumn[E, T](IDStrategyDBAuto)
}

func SnowflakeIDColumn[E any, T any]() ColumnOption[E, T] {
	return WithIDStrategyColumn[E, T](IDStrategySnowflake)
}

func UUIDIDColumn[E any, T any]() ColumnOption[E, T] {
	return WithIDStrategyColumn[E, T](IDStrategyUUID)
}

func UUIDv7IDColumn[E any, T any]() ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.IDStrategy = IDStrategyUUID
		column.meta.UUIDVersion = "v7"
		return column
	}
}

func UUIDv4IDColumn[E any, T any]() ColumnOption[E, T] {
	return func(column Column[E, T]) Column[E, T] {
		column.meta.IDStrategy = IDStrategyUUID
		column.meta.UUIDVersion = "v4"
		return column
	}
}

func (c Column[E, T]) bindColumn(binding columnBinding) any {
	meta := c.meta
	meta.Name = binding.meta.Name
	meta.Table = binding.meta.Table
	meta.Alias = binding.meta.Alias
	meta.FieldName = binding.meta.FieldName
	if meta.GoType == nil {
		meta.GoType = binding.meta.GoType
	}
	if meta.SQLType == "" {
		meta.SQLType = binding.meta.SQLType
	}
	meta.PrimaryKey = meta.PrimaryKey || binding.meta.PrimaryKey
	if meta.IDStrategy == IDStrategyUnset {
		meta.AutoIncrement = meta.AutoIncrement || binding.meta.AutoIncrement
	} else {
		meta.AutoIncrement = meta.IDStrategy == IDStrategyDBAuto
	}
	meta.Nullable = meta.Nullable || binding.meta.Nullable
	meta.Unique = meta.Unique || binding.meta.Unique
	meta.Indexed = meta.Indexed || binding.meta.Indexed
	if meta.DefaultValue == "" {
		meta.DefaultValue = binding.meta.DefaultValue
	}
	if meta.References == nil && binding.meta.References != nil {
		meta.References = new(*binding.meta.References)
	}
	if meta.IDStrategy == IDStrategyUnset {
		meta.IDStrategy = binding.meta.IDStrategy
	}
	if meta.UUIDVersion == "" {
		meta.UUIDVersion = binding.meta.UUIDVersion
	}
	if meta.IDStrategy == IDStrategyUUID && meta.UUIDVersion == "" {
		meta.UUIDVersion = DefaultUUIDVersion
	}
	c.meta = meta
	return c
}

func (c Column[E, T]) expressionNode() {}
func (c Column[E, T]) selectItemNode() {}
func (c Column[E, T]) refNode()        {}

func (Column[E, T]) valueType() reflect.Type {
	return reflect.TypeFor[T]()
}

func (c Column[E, T]) Name() string {
	return c.meta.Name
}

func (c Column[E, T]) TableName() string {
	return c.meta.Table
}

func (c Column[E, T]) TableAlias() string {
	return c.meta.Alias
}

func (c Column[E, T]) FieldName() string {
	return c.meta.FieldName
}

func (c Column[E, T]) Meta() ColumnMeta {
	meta := c.meta
	if meta.References != nil {
		meta.References = new(*meta.References)
	}
	return meta
}

func (c Column[E, T]) IsPrimaryKey() bool {
	return c.meta.PrimaryKey
}

func (c Column[E, T]) IsNullable() bool {
	return c.meta.Nullable
}

func (c Column[E, T]) IsUnique() bool {
	return c.meta.Unique
}

func (c Column[E, T]) IsIndexed() bool {
	return c.meta.Indexed
}

func (c Column[E, T]) DefaultValue() string {
	return c.meta.DefaultValue
}

func (c Column[E, T]) Reference() (ForeignKeyRef, bool) {
	if c.meta.References == nil {
		return ForeignKeyRef{}, false
	}
	return *c.meta.References, true
}

func (c Column[E, T]) Ref() string {
	if c.meta.Alias != "" {
		return c.meta.Alias + "." + c.meta.Name
	}
	return c.meta.Table + "." + c.meta.Name
}

func (c Column[E, T]) Eq(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpEq,
		Right: valueOperand[T]{Value: value},
	}
}

func (c Column[E, T]) EqColumn(other typedColumn[T]) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpEq,
		Right: columnOperand[T]{Column: other},
	}
}

func (c Column[E, T]) Ne(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpNe,
		Right: valueOperand[T]{Value: value},
	}
}

func (c Column[E, T]) Gt(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpGt,
		Right: valueOperand[T]{Value: value},
	}
}

func (c Column[E, T]) Ge(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpGe,
		Right: valueOperand[T]{Value: value},
	}
}

func (c Column[E, T]) Lt(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpLt,
		Right: valueOperand[T]{Value: value},
	}
}

func (c Column[E, T]) Le(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpLe,
		Right: valueOperand[T]{Value: value},
	}
}

func (c Column[E, T]) In(values ...T) Predicate {
	return comparisonPredicate{
		Left: c,
		Op:   OpIn,
		Right: lo.Map(values, func(value T, _ int) any {
			return value
		}),
	}
}

func (c Column[E, T]) InQuery(query *SelectQuery) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpIn,
		Right: subqueryOperand{Query: query},
	}
}

func (c Column[E, T]) IsNull() Predicate {
	return comparisonPredicate{
		Left: c,
		Op:   OpIs,
	}
}

func (c Column[E, T]) IsNotNull() Predicate {
	return comparisonPredicate{
		Left: c,
		Op:   OpIsNot,
	}
}

func (c Column[E, T]) Set(value T) Assignment {
	return columnAssignment[E, T]{
		Column: c,
		Value:  valueOperand[T]{Value: value},
	}
}

func (c Column[E, T]) SetColumn(other typedColumn[T]) Assignment {
	return columnAssignment[E, T]{
		Column: c,
		Value:  columnOperand[T]{Column: other},
	}
}

func (c Column[E, T]) SetExcluded() Assignment {
	return columnAssignment[E, T]{
		Column: c,
		Value:  excludedColumnOperand[T]{Column: c.columnRef()},
	}
}

func (c Column[E, T]) Asc() Order {
	return columnOrder[E, T]{Column: c}
}

func (c Column[E, T]) Desc() Order {
	return columnOrder[E, T]{Column: c, Descending: true}
}

func (c Column[E, T]) table() Table {
	return Table{def: tableDefinition{name: c.meta.Table, alias: c.meta.Alias}}
}

func (c Column[E, T]) columnRef() ColumnMeta {
	return c.meta
}

func (c Column[E, T]) As(alias string) SelectItem {
	return aliasedSelectItem{Item: c, Alias: alias}
}
