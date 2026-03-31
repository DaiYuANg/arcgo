package dbx

import (
	"reflect"
	"slices"
	"strings"

	"github.com/samber/lo"
)

type Table struct {
	def tableDefinition
}

type TableSource interface {
	tableRef() Table
}

type SchemaSource[E any] interface {
	TableSource
	schemaRef() schemaDefinition
}

type schemaBinder interface {
	bindSchema(def schemaDefinition) any
	entityType() reflect.Type
}

type tableDefinition struct {
	name       string
	alias      string
	schemaType reflect.Type
	entityType reflect.Type
}

type schemaDefinition struct {
	table      tableDefinition
	columns    []ColumnMeta
	relations  []RelationMeta
	indexes    []IndexMeta
	primaryKey *PrimaryKeyMeta
	checks     []CheckMeta
}

type Schema[E any] struct {
	def schemaDefinition
}

func (s Schema[E]) bindSchema(def schemaDefinition) any {
	s.def = def
	return s
}

func (Schema[E]) entityType() reflect.Type {
	return reflect.TypeFor[E]()
}

func (s Schema[E]) schemaRef() schemaDefinition {
	return s.def
}

func (s Schema[E]) tableRef() Table {
	return Table{def: s.def.table}
}

func (s Schema[E]) Name() string {
	return s.def.table.name
}

func (s Schema[E]) TableName() string {
	return s.def.table.name
}

func (s Schema[E]) Alias() string {
	return s.def.table.alias
}

func (s Schema[E]) TableAlias() string {
	return s.def.table.alias
}

func (s Schema[E]) Ref() string {
	if s.def.table.alias != "" {
		return s.def.table.alias
	}
	return s.def.table.name
}

func (s Schema[E]) QualifiedName() string {
	if s.def.table.alias == "" || s.def.table.alias == s.def.table.name {
		return s.def.table.name
	}
	return s.def.table.name + " AS " + s.def.table.alias
}

func (s Schema[E]) EntityType() reflect.Type {
	return s.def.table.entityType
}

func (s Schema[E]) Columns() []ColumnMeta {
	return slices.Clone(s.def.columns)
}

func (s Schema[E]) Relations() []RelationMeta {
	return slices.Clone(s.def.relations)
}

func (s Schema[E]) Indexes() []IndexMeta {
	return lo.Map(s.def.indexes, func(item IndexMeta, _ int) IndexMeta {
		return cloneIndexMeta(item)
	})
}

func (s Schema[E]) PrimaryKey() (PrimaryKeyMeta, bool) {
	if s.def.primaryKey == nil {
		return PrimaryKeyMeta{}, false
	}
	return clonePrimaryKeyMeta(*s.def.primaryKey), true
}

func (s Schema[E]) Checks() []CheckMeta {
	return lo.Map(s.def.checks, func(item CheckMeta, _ int) CheckMeta {
		return cloneCheckMeta(item)
	})
}

func (s Schema[E]) ForeignKeys() []ForeignKeyMeta {
	return deriveForeignKeys(s.def)
}

func MustSchema[S any](name string, schema S) S {
	bound, err := bindSchema(name, "", schema)
	if err != nil {
		panic(err)
	}
	return bound
}

func Alias[S TableSource](schema S, alias string) S {
	if strings.TrimSpace(alias) == "" {
		panic("dbx: alias cannot be empty")
	}
	bound, err := bindSchema(schema.tableRef().Name(), alias, schema)
	if err != nil {
		panic(err)
	}
	return bound
}

func (t Table) Name() string {
	return t.def.name
}

func (t Table) TableName() string {
	return t.def.name
}

func (t Table) Alias() string {
	return t.def.alias
}

func (t Table) TableAlias() string {
	return t.def.alias
}

func (t Table) Ref() string {
	if t.def.alias != "" {
		return t.def.alias
	}
	return t.def.name
}

func (t Table) QualifiedName() string {
	if t.def.alias == "" || t.def.alias == t.def.name {
		return t.def.name
	}
	return t.def.name + " AS " + t.def.alias
}

func (t Table) EntityType() reflect.Type {
	return t.def.entityType
}

func (t Table) tableRef() Table {
	return t
}

func NamedTable(name string) Table {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		panic("dbx: named table cannot be empty")
	}
	return Table{def: tableDefinition{name: trimmed}}
}
