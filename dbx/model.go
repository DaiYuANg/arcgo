package dbx

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
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
	Index      int
	Insertable bool
	Updatable  bool
}

type scanPlan struct {
	fields []MappedField
}

type mapperMetadata struct {
	entityType reflect.Type
	fields     collectionx.List[MappedField]
	byColumn   collectionx.Map[string, MappedField]
	scanPlans  collectionx.ConcurrentMap[string, *scanPlan]
}

var structMapperCache = collectionx.NewConcurrentMap[reflect.Type, *mapperMetadata]()

func NewStructMapper[E any]() (StructMapper[E], error) {
	meta, err := getOrBuildStructMapperMetadata[E]()
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

func MustMapper[E any](schema SchemaResource) Mapper[E] {
	mapper, err := NewMapper[E](schema)
	if err != nil {
		panic(err)
	}
	return mapper
}

func NewMapper[E any](schema SchemaResource) (Mapper[E], error) {
	structMapper, err := NewStructMapper[E]()
	if err != nil {
		return Mapper[E]{}, err
	}

	columns := schema.schemaRef().columns
	fields := collectionx.NewListWithCapacity[MappedField](len(columns))
	byColumn := collectionx.NewMapWithCapacity[string, MappedField](len(columns))
	for _, column := range columns {
		field, ok := structMapper.meta.byColumn.Get(column.Name)
		if !ok {
			continue
		}
		fields.Add(field)
		byColumn.Set(column.Name, field)
	}

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

func (m StructMapper[E]) ScanRows(rows *sql.Rows) ([]E, error) {
	if m.meta == nil {
		return nil, ErrNilMapper
	}
	if rows == nil {
		return nil, fmt.Errorf("dbx: rows is nil")
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	plan, err := m.scanPlan(columns)
	if err != nil {
		return nil, err
	}

	items := collectionx.NewList[E]()
	for rows.Next() {
		entity, err := m.scanCurrentRow(rows, plan)
		if err != nil {
			return nil, err
		}
		items.Add(entity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items.Values(), nil
}

func (m Mapper[E]) InsertAssignments(schema SchemaResource, entity *E) ([]Assignment, error) {
	return m.entityAssignments(schema, entity, func(column ColumnMeta, field MappedField) bool {
		if !field.Insertable {
			return false
		}
		return !(column.PrimaryKey && column.AutoIncrement)
	})
}

func (m Mapper[E]) UpdateAssignments(schema SchemaResource, entity *E) ([]Assignment, error) {
	return m.entityAssignments(schema, entity, func(column ColumnMeta, field MappedField) bool {
		if !field.Updatable {
			return false
		}
		return !column.PrimaryKey && !column.AutoIncrement
	})
}

func (m Mapper[E]) PrimaryPredicate(schema SchemaResource, entity *E) (Predicate, error) {
	value, err := m.entityValue(entity)
	if err != nil {
		return nil, err
	}

	for _, column := range schema.schemaRef().columns {
		if !column.PrimaryKey {
			continue
		}
		field, ok := m.byColumn.Get(column.Name)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrPrimaryKeyUnmapped, column.Name)
		}
		return metadataComparisonPredicate{
			left:  column,
			op:    OpEq,
			right: value.Field(field.Index).Interface(),
		}, nil
	}

	return nil, ErrNoPrimaryKey
}

func getOrBuildStructMapperMetadata[E any]() (*mapperMetadata, error) {
	entityType := reflect.TypeFor[E]()
	if cached, ok := structMapperCache.Get(entityType); ok {
		return cached, nil
	}

	mapper, err := buildMapperMetadata(entityType)
	if err != nil {
		return nil, err
	}
	actual, _ := structMapperCache.GetOrStore(entityType, mapper)
	return actual, nil
}

func buildMapperMetadata(entityType reflect.Type) (*mapperMetadata, error) {
	if entityType.Kind() != reflect.Struct {
		return nil, ErrUnsupportedEntity
	}

	fields := collectionx.NewListWithCapacity[MappedField](entityType.NumField())
	byColumn := collectionx.NewMapWithCapacity[string, MappedField](entityType.NumField())
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		if !field.IsExported() {
			continue
		}

		columnName, options := resolveEntityColumn(field)
		if columnName == "" {
			continue
		}

		mapped := MappedField{
			Name:       field.Name,
			Column:     columnName,
			Index:      i,
			Insertable: !options["readonly"] && !options["-insert"] && !options["noinsert"],
			Updatable:  !options["readonly"] && !options["-update"] && !options["noupdate"],
		}
		fields.Add(mapped)
		byColumn.Set(columnName, mapped)
	}

	return &mapperMetadata{
		entityType: entityType,
		fields:     fields,
		byColumn:   byColumn,
		scanPlans:  collectionx.NewConcurrentMapWithCapacity[string, *scanPlan](8),
	}, nil
}

func resolveEntityColumn(field reflect.StructField) (string, map[string]bool) {
	raw := strings.TrimSpace(field.Tag.Get("dbx"))
	if raw == "-" {
		return "", nil
	}
	if raw == "" {
		return toSnakeCase(field.Name), map[string]bool{}
	}

	parts := strings.Split(raw, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		name = toSnakeCase(field.Name)
	}
	options := make(map[string]bool, len(parts)-1)
	for _, option := range parts[1:] {
		trimmed := strings.ToLower(strings.TrimSpace(option))
		if trimmed == "" {
			continue
		}
		options[trimmed] = true
	}
	return name, options
}

func (m StructMapper[E]) scanPlan(columns []string) (*scanPlan, error) {
	signature := scanSignature(columns)
	if cached, ok := m.meta.scanPlans.Get(signature); ok {
		return cached, nil
	}

	fields := collectionx.NewListWithCapacity[MappedField](len(columns))
	for _, column := range columns {
		field, ok := m.meta.byColumn.Get(column)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnmappedColumn, column)
		}
		fields.Add(field)
	}

	plan := &scanPlan{fields: fields.Values()}
	actual, _ := m.meta.scanPlans.GetOrStore(signature, plan)
	return actual, nil
}

func (m StructMapper[E]) scanCurrentRow(rows *sql.Rows, plan *scanPlan) (E, error) {
	value := reflect.New(m.meta.entityType).Elem()
	destinations := make([]any, len(plan.fields))
	for i, field := range plan.fields {
		destinations[i] = value.Field(field.Index).Addr().Interface()
	}

	if err := rows.Scan(destinations...); err != nil {
		var zero E
		return zero, err
	}
	return value.Interface().(E), nil
}

func (m Mapper[E]) entityAssignments(schema SchemaResource, entity *E, include func(column ColumnMeta, field MappedField) bool) ([]Assignment, error) {
	value, err := m.entityValue(entity)
	if err != nil {
		return nil, err
	}

	assignments := collectionx.NewListWithCapacity[Assignment](len(schema.schemaRef().columns))
	for _, column := range schema.schemaRef().columns {
		field, ok := m.byColumn.Get(column.Name)
		if !ok || !include(column, field) {
			continue
		}
		assignments.Add(metadataAssignment{
			meta:  column,
			value: value.Field(field.Index).Interface(),
		})
	}

	return assignments.Values(), nil
}

func (m Mapper[E]) entityValue(entity *E) (reflect.Value, error) {
	if entity == nil {
		return reflect.Value{}, ErrNilEntity
	}
	return reflect.ValueOf(entity).Elem(), nil
}

func scanSignature(columns []string) string {
	return strings.Join(columns, "\x1f")
}
