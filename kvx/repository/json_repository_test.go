package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
)

type mockJSON struct {
	data map[string][]byte
}

func newMockJSON() *mockJSON {
	return &mockJSON{
		data: make(map[string][]byte),
	}
}

func (m *mockJSON) JSONSet(ctx context.Context, key string, path string, value []byte, expiration time.Duration) error {
	m.data[key] = append([]byte(nil), value...)
	return nil
}

func (m *mockJSON) JSONGet(ctx context.Context, key string, path string) ([]byte, error) {
	if value, ok := m.data[key]; ok {
		return append([]byte(nil), value...), nil
	}
	return nil, nil
}

func (m *mockJSON) JSONSetField(ctx context.Context, key string, path string, value []byte) error {
	current, ok := m.data[key]
	if !ok {
		return kvx.ErrNil
	}

	var document map[string]any
	if err := json.Unmarshal(current, &document); err != nil {
		return err
	}

	var fieldValue any
	if err := json.Unmarshal(value, &fieldValue); err != nil {
		return err
	}

	document[extractFieldNameFromPath(path)] = fieldValue

	encoded, err := json.Marshal(document)
	if err != nil {
		return err
	}

	m.data[key] = encoded
	return nil
}

func (m *mockJSON) JSONGetField(ctx context.Context, key string, path string) ([]byte, error) {
	current, ok := m.data[key]
	if !ok {
		return nil, nil
	}

	var document map[string]json.RawMessage
	if err := json.Unmarshal(current, &document); err != nil {
		return nil, err
	}

	return document[extractFieldNameFromPath(path)], nil
}

func (m *mockJSON) JSONDelete(ctx context.Context, key string, path string) error {
	delete(m.data, key)
	return nil
}

func TestJSONRepository_ExistsBatch_UsesKVExists(t *testing.T) {
	ctx := context.Background()
	client := newMockJSON()
	kv := newMockKV()
	kv.data["user:user1"] = []byte("exists")
	kv.data["user:user3"] = []byte("exists")

	repo := NewJSONRepository[TestUser](client, kv, "user")

	results, err := repo.ExistsBatch(ctx, []string{"user1", "user2", "user3"})
	if err != nil {
		t.Fatalf("ExistsBatch failed: %v", err)
	}

	expected := map[string]bool{
		"user1": true,
		"user2": false,
		"user3": true,
	}

	for id, want := range expected {
		if results[id] != want {
			t.Fatalf("expected %s existence to be %v, got %v", id, want, results[id])
		}
	}
}

func TestJSONRepository_FindAll_ScansAllPagesAndDeduplicates(t *testing.T) {
	ctx := context.Background()
	client := newMockJSON()
	kv := newMockKV()
	kv.scanPages = [][]string{
		{"user:user1", "user:user2"},
		{"user:user2", "user:user3"},
	}
	repo := NewJSONRepository[TestUser](client, kv, "user")

	for _, user := range []*TestUser{
		{ID: "user1", Name: "John Doe", Email: "john@example.com", Age: 30},
		{ID: "user2", Name: "Jane Doe", Email: "jane@example.com", Age: 25},
		{ID: "user3", Name: "Bob Doe", Email: "bob@example.com", Age: 40},
	} {
		payload, err := json.Marshal(user)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}
		client.data["user:"+user.ID] = payload
	}

	results, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 unique results, got %d", len(results))
	}

	ids := map[string]bool{}
	for _, result := range results {
		ids[result.ID] = true
	}

	for _, id := range []string{"user1", "user2", "user3"} {
		if !ids[id] {
			t.Fatalf("Expected result for %s", id)
		}
	}
}

func TestJSONRepository_Count_ScansAllPagesAndDeduplicates(t *testing.T) {
	ctx := context.Background()
	client := newMockJSON()
	kv := newMockKV()
	kv.scanPages = [][]string{
		{"user:user1", "user:user2"},
		{"user:user2", "user:user3"},
	}
	repo := NewJSONRepository[TestUser](client, kv, "user")

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if count != 3 {
		t.Fatalf("Expected count 3, got %d", count)
	}
}
