package interval

import common "github.com/DaiYuANg/archgo/collectionx/internal"

// ToJSON serializes normalized ranges to JSON.
func (s *RangeSet[T]) ToJSON() ([]byte, error) {
	return common.MarshalJSONValue(s.Ranges())
}

// MarshalJSON implements json.Marshaler.
func (s *RangeSet[T]) MarshalJSON() ([]byte, error) {
	return common.ForwardToJSON(s.ToJSON)
}

// String implements fmt.Stringer.
func (s *RangeSet[T]) String() string {
	return common.StringFromToJSON(s.ToJSON, "[]")
}

// ToJSON serializes range-map entries to JSON.
func (m *RangeMap[T, V]) ToJSON() ([]byte, error) {
	return common.MarshalJSONValue(m.Entries())
}

// MarshalJSON implements json.Marshaler.
func (m *RangeMap[T, V]) MarshalJSON() ([]byte, error) {
	return common.ForwardToJSON(m.ToJSON)
}

// String implements fmt.Stringer.
func (m *RangeMap[T, V]) String() string {
	return common.StringFromToJSON(m.ToJSON, "[]")
}
