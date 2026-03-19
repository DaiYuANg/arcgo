package dbx

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/DaiYuANg/arcgo/collectionx"
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
	items := make([]ColumnMeta, len(s.def.columns))
	copy(items, s.def.columns)
	return items
}

func (s Schema[E]) Relations() []RelationMeta {
	items := make([]RelationMeta, len(s.def.relations))
	copy(items, s.def.relations)
	return items
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

func bindSchema[S any](name, alias string, schema S) (S, error) {
	value := reflect.ValueOf(&schema).Elem()
	if value.Kind() != reflect.Struct {
		return schema, fmt.Errorf("dbx: schema must be a struct")
	}

	var binder schemaBinder
	var binderField reflect.Value
	schemaType := value.Type()
	defTable := tableDefinition{
		name:       strings.TrimSpace(name),
		alias:      strings.TrimSpace(alias),
		schemaType: schemaType,
	}
	columns := collectionx.NewListWithCapacity[ColumnMeta](value.NumField())
	relations := collectionx.NewListWithCapacity[RelationMeta](value.NumField())
	indexes := collectionx.NewListWithCapacity[IndexMeta](value.NumField())
	checks := collectionx.NewListWithCapacity[CheckMeta](value.NumField())
	var primaryKey *PrimaryKeyMeta

	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		fieldType := schemaType.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		if candidate, ok := fieldValue.Interface().(schemaBinder); ok {
			binder = candidate
			binderField = fieldValue
			defTable.entityType = candidate.entityType()
			continue
		}

		if candidate, ok := fieldValue.Interface().(columnBinder); ok {
			meta := resolveColumnMeta(defTable, fieldType, fieldValue.Interface())
			fieldValue.Set(reflect.ValueOf(candidate.bindColumn(columnBinding{meta: meta})))
			columns.Add(meta)
			continue
		}

		if candidate, ok := fieldValue.Interface().(relationBinder); ok {
			meta := resolveRelationMeta(defTable, fieldType, candidate)
			fieldValue.Set(reflect.ValueOf(candidate.bindRelation(relationBinding{meta: meta})))
			relations.Add(meta)
			continue
		}

		if candidate, ok := fieldValue.Interface().(constraintBinder); ok {
			binding, bindErr := resolveConstraintBinding(defTable, fieldType, fieldValue.Interface())
			if bindErr != nil {
				return schema, bindErr
			}
			fieldValue.Set(reflect.ValueOf(candidate.bindConstraint(binding)))
			indexes.MergeSlice(binding.indexes)
			if binding.primaryKey != nil {
				copyPrimaryKey := clonePrimaryKeyMeta(*binding.primaryKey)
				primaryKey = &copyPrimaryKey
			}
			if binding.check != nil {
				checks.Add(*binding.check)
			}
		}
	}

	if binder == nil {
		return schema, fmt.Errorf("dbx: schema %s must embed dbx.Schema[T]", schemaType.Name())
	}

	binderField.Set(reflect.ValueOf(binder.bindSchema(schemaDefinition{
		table:      defTable,
		columns:    columns.Values(),
		relations:  relations.Values(),
		indexes:    indexes.Values(),
		primaryKey: primaryKey,
		checks:     checks.Values(),
	})))
	return schema, nil
}

func resolveColumnMeta(def tableDefinition, field reflect.StructField, value any) ColumnMeta {
	name, options := resolveTagNameAndOptions(field)
	meta := ColumnMeta{
		Name:          name,
		Table:         def.name,
		Alias:         def.alias,
		FieldName:     field.Name,
		GoType:        resolveColumnGoType(value),
		SQLType:       optionValue(options, "type"),
		PrimaryKey:    optionEnabled(options, "pk"),
		AutoIncrement: optionEnabled(options, "auto") || optionEnabled(options, "autoincrement"),
		Nullable:      optionEnabled(options, "nullable") || optionEnabled(options, "null"),
		Unique:        optionEnabled(options, "unique"),
		Indexed:       optionEnabled(options, "index") || optionEnabled(options, "indexed"),
		DefaultValue:  optionValue(options, "default"),
	}

	if refValue := optionValue(options, "ref"); refValue != "" {
		targetTable, targetColumn, ok := splitReference(refValue)
		if ok {
			meta.References = &ForeignKeyRef{
				TargetTable:  targetTable,
				TargetColumn: targetColumn,
				OnDelete:     parseReferentialAction(optionValue(options, "ondelete")),
				OnUpdate:     parseReferentialAction(optionValue(options, "onupdate")),
			}
		}
	}

	return meta
}

func resolveColumnGoType(value any) reflect.Type {
	reporter, ok := value.(columnTypeReporter)
	if !ok {
		return nil
	}
	return reporter.valueType()
}

func resolveRelationMeta(def tableDefinition, field reflect.StructField, binder relationBinder) RelationMeta {
	options := parseTagOptions(field.Tag.Get("rel"))
	name := optionValue(options, "name")
	if name == "" {
		name = toSnakeCase(field.Name)
	}
	return RelationMeta{
		Name:                name,
		FieldName:           field.Name,
		Kind:                binder.relationKind(),
		SourceTable:         def.name,
		SourceAlias:         def.alias,
		TargetTable:         optionValue(options, "table"),
		LocalColumn:         optionValue(options, "local"),
		TargetColumn:        optionValue(options, "target"),
		ThroughTable:        optionValue(options, "join"),
		ThroughLocalColumn:  optionValue(options, "join_local"),
		ThroughTargetColumn: optionValue(options, "join_target"),
		TargetType:          binder.targetType(),
	}
}

func resolveColumnName(field reflect.StructField) string {
	for _, key := range []string{"column", "dbx", "json"} {
		raw := strings.TrimSpace(field.Tag.Get(key))
		if raw == "" || raw == "-" {
			continue
		}

		name := strings.TrimSpace(strings.Split(raw, ",")[0])
		if name != "" && name != "-" {
			return name
		}
	}

	return toSnakeCase(field.Name)
}

func resolveTagNameAndOptions(field reflect.StructField) (string, map[string]string) {
	raw := strings.TrimSpace(field.Tag.Get("dbx"))
	if raw != "" && raw != "-" {
		parts := strings.Split(raw, ",")
		name := strings.TrimSpace(parts[0])
		if name == "" {
			name = toSnakeCase(field.Name)
		}
		options := make(map[string]string, len(parts)-1)
		for _, part := range parts[1:] {
			key, value := splitTagOption(part)
			if key != "" {
				options[key] = value
			}
		}
		return name, options
	}

	return resolveColumnName(field), map[string]string{}
}

func parseTagOptions(raw string) map[string]string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "-" {
		return map[string]string{}
	}
	parts := strings.Split(trimmed, ",")
	options := make(map[string]string, len(parts))
	for _, part := range parts {
		key, value := splitTagOption(part)
		if key != "" {
			options[key] = value
		}
	}
	return options
}

func splitTagOption(raw string) (string, string) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", ""
	}
	if key, value, ok := strings.Cut(trimmed, "="); ok {
		return strings.TrimSpace(key), strings.TrimSpace(value)
	}
	return trimmed, "true"
}

func optionEnabled(options map[string]string, key string) bool {
	value, ok := options[strings.ToLower(key)]
	if !ok {
		return false
	}
	trimmed := strings.TrimSpace(value)
	return trimmed == "" || trimmed == "true"
}

func optionValue(options map[string]string, key string) string {
	return strings.TrimSpace(options[strings.ToLower(key)])
}

func splitReference(input string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(input), ".")
	if len(parts) != 2 {
		return "", "", false
	}
	table := strings.TrimSpace(parts[0])
	column := strings.TrimSpace(parts[1])
	if table == "" || column == "" {
		return "", "", false
	}
	return table, column, true
}

func parseReferentialAction(input string) ReferentialAction {
	switch strings.ToUpper(strings.TrimSpace(input)) {
	case string(ReferentialCascade):
		return ReferentialCascade
	case string(ReferentialSetNull):
		return ReferentialSetNull
	case string(ReferentialSetDefault):
		return ReferentialSetDefault
	case string(ReferentialRestrict):
		return ReferentialRestrict
	case string(ReferentialNoAction):
		return ReferentialNoAction
	default:
		return ""
	}
}

func toSnakeCase(input string) string {
	if input == "" {
		return ""
	}

	var out strings.Builder
	out.Grow(len(input) + 4)

	for index, r := range input {
		if unicode.IsUpper(r) {
			if index > 0 {
				prev := rune(input[index-1])
				if prev != '_' && (!unicode.IsUpper(prev) || (index+1 < len(input) && unicode.IsLower(rune(input[index+1])))) {
					out.WriteByte('_')
				}
			}
			out.WriteRune(unicode.ToLower(r))
			continue
		}
		out.WriteRune(r)
	}

	return out.String()
}
