package mapping

import (
	"fmt"

	common "github.com/DaiYuANg/arcgo/collectionx/internal"
)

// ToJSON serializes map entries to JSON.
func (m *Map[K, V]) ToJSON() ([]byte, error) {
	return marshalMappingJSON(m.All(), "map")
}

// MarshalJSON implements json.Marshaler.
func (m *Map[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "map")
}

// String implements fmt.Stringer.
func (m *Map[K, V]) String() string {
	return common.StringFromToJSON(m.ToJSON, "{}")
}

// ToJSON serializes concurrent map entries to JSON.
func (m *ConcurrentMap[K, V]) ToJSON() ([]byte, error) {
	return marshalMappingJSON(m.All(), "concurrent map")
}

// MarshalJSON implements json.Marshaler.
func (m *ConcurrentMap[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "concurrent map")
}

// String implements fmt.Stringer.
func (m *ConcurrentMap[K, V]) String() string {
	return common.StringFromToJSON(m.ToJSON, "{}")
}

// ToJSON serializes sharded concurrent map entries to JSON.
func (m *ShardedConcurrentMap[K, V]) ToJSON() ([]byte, error) {
	return marshalMappingJSON(m.All(), "sharded concurrent map")
}

// MarshalJSON implements json.Marshaler.
func (m *ShardedConcurrentMap[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "sharded concurrent map")
}

// String implements fmt.Stringer.
func (m *ShardedConcurrentMap[K, V]) String() string {
	return common.StringFromToJSON(m.ToJSON, "{}")
}

// ToJSON serializes bidirectional map entries to JSON.
func (m *BiMap[K, V]) ToJSON() ([]byte, error) {
	return marshalMappingJSON(m.All(), "bimap")
}

// MarshalJSON implements json.Marshaler.
func (m *BiMap[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "bimap")
}

// String implements fmt.Stringer.
func (m *BiMap[K, V]) String() string {
	return common.StringFromToJSON(m.ToJSON, "{}")
}

// ToJSON serializes ordered map entries to JSON.
func (m *OrderedMap[K, V]) ToJSON() ([]byte, error) {
	return marshalMappingJSON(m.All(), "ordered map")
}

// MarshalJSON implements json.Marshaler.
func (m *OrderedMap[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "ordered map")
}

// String implements fmt.Stringer.
func (m *OrderedMap[K, V]) String() string {
	return common.StringFromToJSON(m.ToJSON, "{}")
}

// ToJSON serializes multimap entries to JSON.
func (m *MultiMap[K, V]) ToJSON() ([]byte, error) {
	return marshalMappingJSON(m.All(), "multimap")
}

// MarshalJSON implements json.Marshaler.
func (m *MultiMap[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "multimap")
}

// String implements fmt.Stringer.
func (m *MultiMap[K, V]) String() string {
	return common.StringFromToJSON(m.ToJSON, "{}")
}

// ToJSON serializes concurrent multimap entries to JSON.
func (m *ConcurrentMultiMap[K, V]) ToJSON() ([]byte, error) {
	return marshalMappingJSON(m.All(), "concurrent multimap")
}

// MarshalJSON implements json.Marshaler.
func (m *ConcurrentMultiMap[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "concurrent multimap")
}

// String implements fmt.Stringer.
func (m *ConcurrentMultiMap[K, V]) String() string {
	return common.StringFromToJSON(m.ToJSON, "{}")
}

// ToJSON serializes table cells to JSON.
func (t *Table[R, C, V]) ToJSON() ([]byte, error) {
	return marshalMappingJSON(t.All(), "table")
}

// MarshalJSON implements json.Marshaler.
func (t *Table[R, C, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(t.ToJSON, "table")
}

// String implements fmt.Stringer.
func (t *Table[R, C, V]) String() string {
	return common.StringFromToJSON(t.ToJSON, "{}")
}

// ToJSON serializes concurrent table cells to JSON.
func (t *ConcurrentTable[R, C, V]) ToJSON() ([]byte, error) {
	return marshalMappingJSON(t.All(), "concurrent table")
}

// MarshalJSON implements json.Marshaler.
func (t *ConcurrentTable[R, C, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(t.ToJSON, "concurrent table")
}

// String implements fmt.Stringer.
func (t *ConcurrentTable[R, C, V]) String() string {
	return common.StringFromToJSON(t.ToJSON, "{}")
}

func marshalMappingJSON(value any, kind string) ([]byte, error) {
	data, err := common.MarshalJSONValue(value)
	if err != nil {
		return nil, fmt.Errorf("marshal %s json: %w", kind, err)
	}
	return data, nil
}

func forwardMappingJSON(toJSON func() ([]byte, error), kind string) ([]byte, error) {
	data, err := common.ForwardToJSON(toJSON)
	if err != nil {
		return nil, fmt.Errorf("marshal %s: %w", kind, err)
	}
	return data, nil
}
