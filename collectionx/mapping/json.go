//revive:disable:file-length-limit JSON mapping helpers are kept together to preserve the collection API surface.

package mapping

import (
	"encoding/json"
	"fmt"
	"slices"

	common "github.com/DaiYuANg/arcgo/collectionx/internal"
)

// ToJSON serializes map entries to JSON.
func (m *Map[K, V]) ToJSON() ([]byte, error) {
	if m != nil && !m.jsonDirty && m.jsonCache != nil {
		return slices.Clone(m.jsonCache), nil
	}

	var (
		data []byte
		err  error
	)
	if m == nil || len(m.items) == 0 {
		data, err = marshalMappingJSON(map[K]V{}, "map")
	} else {
		data, err = marshalMappingJSON(m.items, "map")
	}
	if err != nil {
		return nil, err
	}
	if m != nil {
		m.cacheSerializationData(data)
	}
	return slices.Clone(data), nil
}

// MarshalJSON implements json.Marshaler.
func (m *Map[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "map")
}

// String implements fmt.Stringer.
func (m *Map[K, V]) String() string {
	if m != nil && !m.jsonDirty && m.stringCache != "" {
		return m.stringCache
	}
	data, err := m.ToJSON()
	return common.JSONResultString(data, err, "{}")
}

// ToJSON serializes concurrent map entries to JSON.
func (m *ConcurrentMap[K, V]) ToJSON() ([]byte, error) {
	if m == nil {
		return marshalMappingJSON(map[K]V{}, "concurrent map")
	}

	m.mu.RLock()
	if !m.jsonDirty && m.jsonCache != nil {
		data := slices.Clone(m.jsonCache)
		m.mu.RUnlock()
		return data, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.jsonDirty && m.jsonCache != nil {
		return slices.Clone(m.jsonCache), nil
	}

	var (
		data []byte
		err  error
	)
	if m.core == nil || len(m.core.items) == 0 {
		data, err = marshalMappingJSON(map[K]V{}, "concurrent map")
	} else {
		data, err = marshalMappingJSON(m.core.items, "concurrent map")
	}
	if err != nil {
		return nil, err
	}
	m.jsonCache = data
	m.stringCache = string(data)
	m.jsonDirty = false
	return slices.Clone(data), nil
}

// MarshalJSON implements json.Marshaler.
func (m *ConcurrentMap[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "concurrent map")
}

// String implements fmt.Stringer.
func (m *ConcurrentMap[K, V]) String() string {
	if m == nil {
		return "{}"
	}
	m.mu.RLock()
	if !m.jsonDirty && m.stringCache != "" {
		value := m.stringCache
		m.mu.RUnlock()
		return value
	}
	m.mu.RUnlock()
	data, err := m.ToJSON()
	return common.JSONResultString(data, err, "{}")
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
	if m == nil || len(m.kv.items) == 0 {
		return marshalMappingJSON(map[K]V{}, "bimap")
	}
	return marshalMappingJSON(m.kv.items, "bimap")
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
	if m != nil && !m.jsonDirty && m.jsonCache != nil {
		return slices.Clone(m.jsonCache), nil
	}
	var (
		data []byte
		err  error
	)
	if m == nil || len(m.items.items) == 0 {
		data, err = marshalMappingJSON(map[K]V{}, "ordered map")
	} else {
		data, err = marshalMappingJSON(m.items.items, "ordered map")
	}
	if err != nil {
		return nil, err
	}
	if m != nil {
		m.cacheSerializationData(data)
	}
	return slices.Clone(data), nil
}

// MarshalJSON implements json.Marshaler.
func (m *OrderedMap[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "ordered map")
}

// String implements fmt.Stringer.
func (m *OrderedMap[K, V]) String() string {
	if m != nil && !m.jsonDirty && m.stringCache != "" {
		return m.stringCache
	}
	data, err := m.ToJSON()
	return common.JSONResultString(data, err, "{}")
}

// ToJSON serializes multimap entries to JSON.
func (m *MultiMap[K, V]) ToJSON() ([]byte, error) {
	if m != nil && !m.jsonDirty && m.jsonCache != nil {
		return slices.Clone(m.jsonCache), nil
	}
	data, err := marshalMappingJSON(m.All(), "multimap")
	if err != nil {
		return nil, err
	}
	if m != nil {
		m.cacheSerializationData(data)
	}
	return slices.Clone(data), nil
}

// MarshalJSON implements json.Marshaler.
func (m *MultiMap[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "multimap")
}

// String implements fmt.Stringer.
func (m *MultiMap[K, V]) String() string {
	if m != nil && !m.jsonDirty && m.stringCache != "" {
		return m.stringCache
	}
	data, err := m.ToJSON()
	return common.JSONResultString(data, err, "{}")
}

// ToJSON serializes concurrent multimap entries to JSON.
func (m *ConcurrentMultiMap[K, V]) ToJSON() ([]byte, error) {
	if m == nil {
		return marshalMappingJSON(map[K][]V{}, "concurrent multimap")
	}

	m.mu.RLock()
	if !m.jsonDirty && m.jsonCache != nil {
		data := slices.Clone(m.jsonCache)
		m.mu.RUnlock()
		return data, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.jsonDirty && m.jsonCache != nil {
		return slices.Clone(m.jsonCache), nil
	}

	var (
		data []byte
		err  error
	)
	if m.core == nil || m.core.items.Len() == 0 {
		data, err = marshalMappingJSON(map[K][]V{}, "concurrent multimap")
	} else {
		data, err = marshalMappingJSON(m.core.All(), "concurrent multimap")
	}
	if err != nil {
		return nil, err
	}
	m.jsonCache = data
	m.stringCache = string(data)
	m.jsonDirty = false
	return slices.Clone(data), nil
}

// MarshalJSON implements json.Marshaler.
func (m *ConcurrentMultiMap[K, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(m.ToJSON, "concurrent multimap")
}

// String implements fmt.Stringer.
func (m *ConcurrentMultiMap[K, V]) String() string {
	if m == nil {
		return "{}"
	}
	m.mu.RLock()
	if !m.jsonDirty && m.stringCache != "" {
		value := m.stringCache
		m.mu.RUnlock()
		return value
	}
	m.mu.RUnlock()
	data, err := m.ToJSON()
	return common.JSONResultString(data, err, "{}")
}

// ToJSON serializes table cells to JSON.
func (t *Table[R, C, V]) ToJSON() ([]byte, error) {
	if t != nil && !t.jsonDirty && t.jsonCache != nil {
		return slices.Clone(t.jsonCache), nil
	}
	data, err := marshalMappingJSON(t.All(), "table")
	if err != nil {
		return nil, err
	}
	if t != nil {
		t.cacheSerializationData(data)
	}
	return slices.Clone(data), nil
}

// MarshalJSON implements json.Marshaler.
func (t *Table[R, C, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(t.ToJSON, "table")
}

// String implements fmt.Stringer.
func (t *Table[R, C, V]) String() string {
	if t != nil && !t.jsonDirty && t.stringCache != "" {
		return t.stringCache
	}
	data, err := t.ToJSON()
	return common.JSONResultString(data, err, "{}")
}

// ToJSON serializes concurrent table cells to JSON.
func (t *ConcurrentTable[R, C, V]) ToJSON() ([]byte, error) {
	if t == nil {
		return marshalMappingJSON(map[R]map[C]V{}, "concurrent table")
	}

	t.mu.RLock()
	if !t.jsonDirty && t.jsonCache != nil {
		data := slices.Clone(t.jsonCache)
		t.mu.RUnlock()
		return data, nil
	}
	t.mu.RUnlock()

	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.jsonDirty && t.jsonCache != nil {
		return slices.Clone(t.jsonCache), nil
	}

	var (
		data []byte
		err  error
	)
	if t.core == nil || t.core.data.Len() == 0 {
		data, err = marshalMappingJSON(map[R]map[C]V{}, "concurrent table")
	} else {
		data, err = marshalMappingJSON(t.core.All(), "concurrent table")
	}
	if err != nil {
		return nil, err
	}
	t.jsonCache = data
	t.stringCache = string(data)
	t.jsonDirty = false
	return slices.Clone(data), nil
}

// MarshalJSON implements json.Marshaler.
func (t *ConcurrentTable[R, C, V]) MarshalJSON() ([]byte, error) {
	return forwardMappingJSON(t.ToJSON, "concurrent table")
}

// String implements fmt.Stringer.
func (t *ConcurrentTable[R, C, V]) String() string {
	if t == nil {
		return "{}"
	}
	t.mu.RLock()
	if !t.jsonDirty && t.stringCache != "" {
		value := t.stringCache
		t.mu.RUnlock()
		return value
	}
	t.mu.RUnlock()
	data, err := t.ToJSON()
	return common.JSONResultString(data, err, "{}")
}

func marshalMappingJSON(value any, kind string) ([]byte, error) {
	data, err := json.Marshal(value)
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
