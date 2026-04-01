package repository_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
)

func newMockJSON() *mockJSON {
	return &mockJSON{
		data: make(map[string][]byte),
	}
}

func (m *mockJSON) JSONSet(_ context.Context, key, _ string, value []byte, _ time.Duration) error {
	m.data[key] = append([]byte(nil), value...)
	return nil
}

func (m *mockJSON) JSONGet(_ context.Context, key, _ string) ([]byte, error) {
	if value, ok := m.data[key]; ok {
		return append([]byte(nil), value...), nil
	}

	return nil, nil
}

func (m *mockJSON) JSONSetField(_ context.Context, key, path string, value []byte) error {
	current, ok := m.data[key]
	if !ok {
		return kvx.ErrNil
	}

	var document map[string]any
	if err := json.Unmarshal(current, &document); err != nil {
		return fmt.Errorf("unmarshal JSON document: %w", err)
	}

	var fieldValue any
	if err := json.Unmarshal(value, &fieldValue); err != nil {
		return fmt.Errorf("unmarshal JSON field value: %w", err)
	}

	document[fieldNameFromPath(path)] = fieldValue

	encoded, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("marshal JSON document: %w", err)
	}

	m.data[key] = encoded
	return nil
}

func (m *mockJSON) JSONGetField(_ context.Context, key, path string) ([]byte, error) {
	current, ok := m.data[key]
	if !ok {
		return nil, nil
	}

	var document map[string]json.RawMessage
	if err := json.Unmarshal(current, &document); err != nil {
		return nil, fmt.Errorf("unmarshal JSON field map: %w", err)
	}

	return document[fieldNameFromPath(path)], nil
}

func (m *mockJSON) JSONDelete(_ context.Context, key, _ string) error {
	delete(m.data, key)
	return nil
}
