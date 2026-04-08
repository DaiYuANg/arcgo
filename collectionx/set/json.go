package set

import (
	"fmt"
	"slices"

	common "github.com/DaiYuANg/arcgo/collectionx/internal"
)

// ToJSON serializes set values to JSON.
func (s *Set[T]) ToJSON() ([]byte, error) {
	if s != nil && !s.jsonDirty && s.jsonCache != nil {
		return slices.Clone(s.jsonCache), nil
	}
	data, err := marshalSetJSON("set", s.Values())
	if err != nil {
		return nil, err
	}
	if s != nil {
		s.cacheSerializationData(data)
	}
	return slices.Clone(data), nil
}

// MarshalJSON implements json.Marshaler.
func (s *Set[T]) MarshalJSON() ([]byte, error) {
	return forwardSetJSON("set", s.ToJSON)
}

// String implements fmt.Stringer.
func (s *Set[T]) String() string {
	if s != nil && !s.jsonDirty && s.stringCache != "" {
		return s.stringCache
	}
	data, err := s.ToJSON()
	return common.JSONResultString(data, err, "[]")
}

// ToJSON serializes concurrent set values to JSON.
func (s *ConcurrentSet[T]) ToJSON() ([]byte, error) {
	if s == nil {
		return marshalSetJSON("concurrent set", []T(nil))
	}

	s.mu.RLock()
	if !s.jsonDirty && s.jsonCache != nil {
		data := slices.Clone(s.jsonCache)
		s.mu.RUnlock()
		return data, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.jsonDirty && s.jsonCache != nil {
		return slices.Clone(s.jsonCache), nil
	}

	var values []T
	if s.core != nil {
		values = s.core.Values()
	}
	data, err := marshalSetJSON("concurrent set", values)
	if err != nil {
		return nil, err
	}
	s.jsonCache = data
	s.stringCache = string(data)
	s.jsonDirty = false
	return slices.Clone(data), nil
}

// MarshalJSON implements json.Marshaler.
func (s *ConcurrentSet[T]) MarshalJSON() ([]byte, error) {
	return forwardSetJSON("concurrent set", s.ToJSON)
}

// String implements fmt.Stringer.
func (s *ConcurrentSet[T]) String() string {
	if s == nil {
		return "[]"
	}
	s.mu.RLock()
	if !s.jsonDirty && s.stringCache != "" {
		value := s.stringCache
		s.mu.RUnlock()
		return value
	}
	s.mu.RUnlock()
	data, err := s.ToJSON()
	return common.JSONResultString(data, err, "[]")
}

// ToJSON serializes multiset counts to JSON.
func (s *MultiSet[T]) ToJSON() ([]byte, error) {
	return marshalSetJSON("multiset", s.AllCounts())
}

// MarshalJSON implements json.Marshaler.
func (s *MultiSet[T]) MarshalJSON() ([]byte, error) {
	return forwardSetJSON("multiset", s.ToJSON)
}

// String implements fmt.Stringer.
func (s *MultiSet[T]) String() string {
	return common.StringFromToJSON(s.ToJSON, "{}")
}

// ToJSON serializes ordered set values to JSON.
func (s *OrderedSet[T]) ToJSON() ([]byte, error) {
	if s != nil && !s.jsonDirty && s.jsonCache != nil {
		return slices.Clone(s.jsonCache), nil
	}
	data, err := marshalSetJSON("ordered set", s.Values())
	if err != nil {
		return nil, err
	}
	if s != nil {
		s.cacheSerializationData(data)
	}
	return slices.Clone(data), nil
}

// MarshalJSON implements json.Marshaler.
func (s *OrderedSet[T]) MarshalJSON() ([]byte, error) {
	return forwardSetJSON("ordered set", s.ToJSON)
}

// String implements fmt.Stringer.
func (s *OrderedSet[T]) String() string {
	if s != nil && !s.jsonDirty && s.stringCache != "" {
		return s.stringCache
	}
	data, err := s.ToJSON()
	return common.JSONResultString(data, err, "[]")
}

func marshalSetJSON[T any](kind string, value T) ([]byte, error) {
	data, err := common.MarshalJSONValue(value)
	if err != nil {
		return nil, fmt.Errorf("marshal %s JSON: %w", kind, err)
	}

	return data, nil
}

func forwardSetJSON(kind string, fn func() ([]byte, error)) ([]byte, error) {
	data, err := common.ForwardToJSON(fn)
	if err != nil {
		return nil, fmt.Errorf("marshal %s JSON: %w", kind, err)
	}

	return data, nil
}
