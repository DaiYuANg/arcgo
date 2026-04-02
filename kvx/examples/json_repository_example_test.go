package examples_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/repository"
)

type exampleJSONBackend struct {
	data map[string][]byte
}

func newExampleJSONBackend() *exampleJSONBackend {
	return &exampleJSONBackend{data: map[string][]byte{}}
}
func (m *exampleJSONBackend) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := m.data[key]
	if !ok {
		return nil, kvx.ErrNil
	}
	return v, nil
}
func (m *exampleJSONBackend) MGet(context.Context, []string) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}
func (m *exampleJSONBackend) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.data[key] = value
	return nil
}
func (m *exampleJSONBackend) MSet(context.Context, map[string][]byte, time.Duration) error {
	return nil
}
func (m *exampleJSONBackend) Delete(_ context.Context, key string) error {
	delete(m.data, key)
	return nil
}
func (m *exampleJSONBackend) DeleteMulti(context.Context, []string) error { return nil }
func (m *exampleJSONBackend) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}
func (m *exampleJSONBackend) ExistsMulti(context.Context, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (m *exampleJSONBackend) Expire(context.Context, string, time.Duration) error { return nil }
func (m *exampleJSONBackend) TTL(context.Context, string) (time.Duration, error)  { return 0, nil }
func (m *exampleJSONBackend) Scan(_ context.Context, _ string, _ uint64, _ int64) (collectionx.List[string], uint64, error) {
	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}
	return collectionx.NewListWithCapacity(len(keys), keys...), 0, nil
}
func (m *exampleJSONBackend) Keys(ctx context.Context, pattern string) (collectionx.List[string], error) {
	keys, _, err := m.Scan(ctx, pattern, 0, 0)
	return keys, err
}
func (m *exampleJSONBackend) JSONSet(_ context.Context, key, _ string, value []byte, _ time.Duration) error {
	m.data[key] = value
	return nil
}
func (m *exampleJSONBackend) JSONGet(ctx context.Context, key, _ string) ([]byte, error) {
	return m.Get(ctx, key)
}
func (m *exampleJSONBackend) JSONSetField(_ context.Context, key, path string, value []byte) error {
	var doc map[string]any
	if err := json.Unmarshal(m.data[key], &doc); err != nil {
		return fmt.Errorf("unmarshal json document %q: %w", key, err)
	}
	doc[path[2:]] = string(value)
	encoded, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal json document %q: %w", key, err)
	}
	m.data[key] = encoded
	return nil
}
func (m *exampleJSONBackend) JSONGetField(context.Context, string, string) ([]byte, error) {
	return nil, nil
}
func (m *exampleJSONBackend) JSONDelete(_ context.Context, key, _ string) error {
	delete(m.data, key)
	return nil
}

func ExampleJSONRepository() {
	backend := newExampleJSONBackend()
	repo := repository.NewJSONRepository[ExampleUser](backend, backend, "json:user")
	if err := repo.Save(context.Background(), &ExampleUser{ID: "u-2", Name: "Bob", Email: "bob@example.com"}); err != nil {
		panic(err)
	}
	entity, err := repo.FindByID(context.Background(), "u-2")
	if err != nil {
		panic(err)
	}
	if _, err := fmt.Println(entity.ID, entity.Email); err != nil {
		panic(err)
	}
	// Output: u-2 bob@example.com
}
