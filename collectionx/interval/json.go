package interval

import (
	"fmt"

	common "github.com/DaiYuANg/arcgo/collectionx/internal"
)

// ToJSON serializes normalized ranges to JSON.
func (s *RangeSet[T]) ToJSON() ([]byte, error) {
	data, err := common.MarshalJSONValue(s.Ranges())
	if err != nil {
		return nil, fmt.Errorf("marshal range set json: %w", err)
	}
	return data, nil
}

// MarshalJSON implements json.Marshaler.
func (s *RangeSet[T]) MarshalJSON() ([]byte, error) {
	data, err := common.ForwardToJSON(s.ToJSON)
	if err != nil {
		return nil, fmt.Errorf("marshal range set: %w", err)
	}
	return data, nil
}

// String implements fmt.Stringer.
func (s *RangeSet[T]) String() string {
	return common.StringFromToJSON(s.ToJSON, "[]")
}

// ToJSON serializes range-map entries to JSON.
func (m *RangeMap[T, V]) ToJSON() ([]byte, error) {
	data, err := common.MarshalJSONValue(m.Entries())
	if err != nil {
		return nil, fmt.Errorf("marshal range map json: %w", err)
	}
	return data, nil
}

// MarshalJSON implements json.Marshaler.
func (m *RangeMap[T, V]) MarshalJSON() ([]byte, error) {
	data, err := common.ForwardToJSON(m.ToJSON)
	if err != nil {
		return nil, fmt.Errorf("marshal range map: %w", err)
	}
	return data, nil
}

// String implements fmt.Stringer.
func (m *RangeMap[T, V]) String() string {
	return common.StringFromToJSON(m.ToJSON, "[]")
}
