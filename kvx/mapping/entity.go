package mapping

import (
	"reflect"
	"strings"
	"sync"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

// FieldTag represents metadata for a struct field.
type FieldTag struct {
	FieldName string // Go struct field name
	Name      string // field name in storage
	Ignored   bool   // whether to ignore this field
	Index     bool   // whether this field is indexed
	IndexName string // custom index name
}

// IndexNameOrDefault returns the effective index field name.
func (f FieldTag) IndexNameOrDefault() string {
	if f.IndexName != "" {
		return f.IndexName
	}
	if f.Name != "" {
		return f.Name
	}
	return f.FieldName
}

// StorageName returns the effective stored field name.
func (f FieldTag) StorageName() string {
	if f.Name != "" {
		return f.Name
	}
	return f.FieldName
}

// EntityMetadata holds metadata for an entity type.
type EntityMetadata struct {
	Type            reflect.Type
	KeyField        string // field name for the entity key/ID
	KeyPrefix       string // prefix for generating keys
	Fields          map[string]FieldTag
	IndexFields     []string // list of indexed field names
	HasExpiration   bool
	ExpirationField string
}

// StorageNames returns all storage field names.
func (m *EntityMetadata) StorageNames() collectionx.List[string] {
	fieldNames := orderedFieldNames(m)
	return collectionx.MapList(fieldNames, func(_ int, fieldName string) string {
		return m.Fields[fieldName].StorageName()
	})
}

// IndexedNames returns all effective indexed field names.
func (m *EntityMetadata) IndexedNames() collectionx.List[string] {
	names := collectionx.NewOrderedSetWithCapacity[string](len(m.IndexFields))
	lo.ForEach(m.IndexFields, func(fieldName string, _ int) {
		field, ok := m.Fields[fieldName]
		if !ok {
			names.Add(fieldName)
			return
		}
		names.Add(field.IndexNameOrDefault())
	})

	indexed := collectionx.NewListWithCapacity[string](names.Len())
	names.Range(func(item string) bool {
		indexed.Add(item)
		return true
	})
	return indexed
}

// ResolveField resolves a struct field, storage field, or index alias into a struct field name and metadata.
func (m *EntityMetadata) ResolveField(name string) (string, FieldTag, bool) {
	if field, ok := m.Fields[name]; ok {
		return name, field, true
	}

	for fieldName, field := range m.Fields {
		if field.StorageName() == name || field.IndexNameOrDefault() == name {
			return fieldName, field, true
		}
	}

	return "", FieldTag{}, false
}

// KeyFieldTag returns metadata for the key field when it is exported through Fields.
func (m *EntityMetadata) KeyFieldTag() (FieldTag, bool) {
	field, ok := m.Fields[m.KeyField]
	return field, ok
}

// SetEntityID fills the key field value from a raw ID string.
func (m *EntityMetadata) SetEntityID(entity any, id string) error {
	if m.KeyField == "" || id == "" {
		return nil
	}

	v := reflect.ValueOf(entity)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrNonPointerValue
	}
	v = v.Elem()

	field := v.FieldByName(m.KeyField)
	if !field.IsValid() || !field.CanSet() {
		return nil
	}

	return setFieldStringValue(field, id)
}

// Schema is the stable object description used by repositories and indexers.
type Schema = EntityMetadata

// TagParser parses struct tags into metadata.
type TagParser struct {
	cache sync.Map // map[reflect.Type]*EntityMetadata
}

// NewTagParser creates a new TagParser.
func NewTagParser() *TagParser {
	return &TagParser{}
}

// Parse parses metadata from a struct type.
func (p *TagParser) Parse(t reflect.Type) (*EntityMetadata, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, ErrNonStructType
	}

	if cached, ok := p.cache.Load(t); ok {
		if metadata, ok := cachedEntityMetadata(cached); ok {
			return metadata, nil
		}
	}

	metadata, err := p.parseStruct(t)
	if err != nil {
		return nil, err
	}
	p.cache.Store(t, metadata)
	return metadata, nil
}

// ParseType parses metadata from an entity instance.
func (p *TagParser) ParseType(entity any) (*EntityMetadata, error) {
	return p.Parse(reflect.TypeOf(entity))
}

func (p *TagParser) parseStruct(t reflect.Type) (*EntityMetadata, error) {
	metadata := newEntityMetadata(t)

	for field := range t.Fields() {
		p.addFieldMetadata(metadata, field)
	}

	if metadata.KeyField == "" {
		return nil, ErrNoKeyFieldDefined
	}

	metadata.IndexFields = lo.Uniq(metadata.IndexFields)
	return metadata, nil
}

func (p *TagParser) parseFieldTag(fieldName, tag string) FieldTag {
	result := FieldTag{FieldName: fieldName}
	parts := strings.Split(tag, ",")

	if len(parts) > 0 {
		name := strings.TrimSpace(parts[0])
		switch name {
		case "-":
			result.Ignored = true
		case "", "id", "key":
			result.Name = name
		default:
			result.Name = name
		}
	}

	lo.ForEach(parts[1:], func(part string, _ int) {
		part = strings.TrimSpace(part)
		switch {
		case part == "omitempty":
			return
		case part == "index":
			result.Index = true
		case strings.HasPrefix(part, "index="):
			result.Index = true
			result.IndexName = strings.TrimPrefix(part, "index=")
		case part == "ignore":
			result.Ignored = true
		}
	})

	return result
}

// GetCached returns cached metadata for a type if available.
func (p *TagParser) GetCached(t reflect.Type) *EntityMetadata {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if cached, ok := p.cache.Load(t); ok {
		metadata, ok := cachedEntityMetadata(cached)
		if ok {
			return metadata
		}
	}
	return nil
}

func newEntityMetadata(t reflect.Type) *EntityMetadata {
	return &EntityMetadata{
		Type:   t,
		Fields: make(map[string]FieldTag),
	}
}

func cachedEntityMetadata(value any) (*EntityMetadata, bool) {
	metadata, ok := value.(*EntityMetadata)
	return metadata, ok
}

func (p *TagParser) addFieldMetadata(metadata *EntityMetadata, field reflect.StructField) {
	if !field.IsExported() {
		return
	}

	tag := field.Tag.Get("kvx")
	if tag == "" {
		return
	}

	fieldTag := p.parseFieldTag(field.Name, tag)
	if fieldTag.Ignored {
		return
	}

	metadata.Fields[field.Name] = fieldTag
	if isKeyField(fieldTag) {
		metadata.KeyField = field.Name
		return
	}
	if fieldTag.Index {
		metadata.IndexFields = lo.Concat(metadata.IndexFields, []string{field.Name})
	}
}

func isKeyField(fieldTag FieldTag) bool {
	return fieldTag.Name == "id" || fieldTag.Name == "key"
}

func orderedFieldNames(m *EntityMetadata) collectionx.List[string] {
	if m == nil || len(m.Fields) == 0 {
		return collectionx.NewList[string]()
	}

	names := collectionx.NewListWithCapacity[string](len(m.Fields))
	lo.ForEach(lo.Entries(m.Fields), func(entry lo.Entry[string, FieldTag], _ int) {
		if entry.Value.Ignored || entry.Key == m.KeyField {
			return
		}
		names.Add(entry.Key)
	})
	return names
}
