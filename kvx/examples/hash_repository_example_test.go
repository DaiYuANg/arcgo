package examples_test

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/DaiYuANg/arcgo/kvx/repository"
)

type ExampleUser struct {
	ID    string `kvx:"id"`
	Name  string `kvx:"name"`
	Email string `kvx:"email,index=email"`
}

type exampleHashKV struct {
	hashes map[string]map[string][]byte
	keys   map[string][]byte
}

func newExampleHashKV() *exampleHashKV {
	return &exampleHashKV{hashes: map[string]map[string][]byte{}, keys: map[string][]byte{}}
}

func (m *exampleHashKV) Get(context.Context, string) ([]byte, error) { return nil, kvx.ErrNil }
func (m *exampleHashKV) MGet(context.Context, []string) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}
func (m *exampleHashKV) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.keys[key] = value
	return nil
}
func (m *exampleHashKV) MSet(context.Context, map[string][]byte, time.Duration) error { return nil }
func (m *exampleHashKV) Delete(_ context.Context, key string) error {
	delete(m.keys, key)
	delete(m.hashes, key)
	return nil
}
func (m *exampleHashKV) DeleteMulti(context.Context, []string) error { return nil }
func (m *exampleHashKV) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.keys[key]
	return ok, nil
}
func (m *exampleHashKV) ExistsMulti(context.Context, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (m *exampleHashKV) Expire(context.Context, string, time.Duration) error { return nil }
func (m *exampleHashKV) TTL(context.Context, string) (time.Duration, error)  { return 0, nil }
func (m *exampleHashKV) Scan(_ context.Context, pattern string, _ uint64, _ int64) (collectionx.List[string], uint64, error) {
	keys := make([]string, 0)
	for key := range m.keys {
		if pattern != "" && pattern[len(pattern)-1] == '*' && len(key) >= len(pattern)-1 && key[:len(pattern)-1] == pattern[:len(pattern)-1] {
			keys = append(keys, key)
		}
	}
	return collectionx.NewListWithCapacity(len(keys), keys...), 0, nil
}
func (m *exampleHashKV) Keys(ctx context.Context, pattern string) (collectionx.List[string], error) {
	keys, _, err := m.Scan(ctx, pattern, 0, 0)
	return keys, err
}
func (m *exampleHashKV) HGet(context.Context, string, string) ([]byte, error) { return nil, kvx.ErrNil }
func (m *exampleHashKV) HMGet(context.Context, string, []string) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}
func (m *exampleHashKV) HSet(_ context.Context, key string, values map[string][]byte) error {
	m.hashes[key] = values
	m.keys[key] = []byte("1")
	return nil
}
func (m *exampleHashKV) HMSet(ctx context.Context, key string, values map[string][]byte) error {
	return m.HSet(ctx, key, values)
}
func (m *exampleHashKV) HGetAll(_ context.Context, key string) (map[string][]byte, error) {
	return m.hashes[key], nil
}
func (m *exampleHashKV) HDel(context.Context, string, ...string) error         { return nil }
func (m *exampleHashKV) HExists(context.Context, string, string) (bool, error) { return false, nil }
func (m *exampleHashKV) HKeys(context.Context, string) (collectionx.List[string], error) {
	return collectionx.NewList[string](), nil
}
func (m *exampleHashKV) HVals(context.Context, string) (collectionx.List[[]byte], error) {
	return collectionx.NewList[[]byte](), nil
}
func (m *exampleHashKV) HLen(context.Context, string) (int64, error)                   { return 0, nil }
func (m *exampleHashKV) HIncrBy(context.Context, string, string, int64) (int64, error) { return 0, nil }

func ExampleHashRepository() {
	backend := newExampleHashKV()
	preset := repository.NewPreset[ExampleUser](
		repository.WithKeyBuilder[ExampleUser](mapping.NewKeyBuilder("demo:user")),
	)

	repo := repository.NewHashRepository[ExampleUser](backend, backend, "user", preset.HashOptions(
		repository.WithHashCodec[ExampleUser](mapping.NewHashCodec(nil)),
	)...)

	if err := repo.Save(context.Background(), &ExampleUser{ID: "u-1", Name: "Alice", Email: "alice@example.com"}); err != nil {
		panic(err)
	}
	entity, err := repo.FindByID(context.Background(), "u-1")
	if err != nil {
		panic(err)
	}
	if _, err := fmt.Println(entity.ID, entity.Name); err != nil {
		panic(err)
	}
	// Output: u-1 Alice
}
