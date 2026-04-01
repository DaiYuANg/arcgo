package repository_test

import (
	"context"
	"maps"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
)

const maxInt = int(^uint(0) >> 1)

type mockHash struct {
	data map[string]map[string][]byte
}

func newMockHash() *mockHash {
	return &mockHash{
		data: make(map[string]map[string][]byte),
	}
}

func (m *mockHash) HGet(_ context.Context, key, field string) ([]byte, error) {
	if hash, ok := m.data[key]; ok {
		if value, ok := hash[field]; ok {
			return value, nil
		}
	}

	return nil, kvx.ErrNil
}

func (m *mockHash) HMGet(_ context.Context, key string, fields []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	if hash, ok := m.data[key]; ok {
		for _, field := range fields {
			if value, ok := hash[field]; ok {
				result[field] = value
			}
		}
	}

	return result, nil
}

func (m *mockHash) HSet(_ context.Context, key string, values map[string][]byte) error {
	if _, ok := m.data[key]; !ok {
		m.data[key] = make(map[string][]byte)
	}

	maps.Copy(m.data[key], values)
	return nil
}

func (m *mockHash) HMSet(ctx context.Context, key string, values map[string][]byte) error {
	return m.HSet(ctx, key, values)
}

func (m *mockHash) HGetAll(_ context.Context, key string) (map[string][]byte, error) {
	if hash, ok := m.data[key]; ok {
		result := make(map[string][]byte, len(hash))
		maps.Copy(result, hash)
		return result, nil
	}

	return make(map[string][]byte), nil
}

func (m *mockHash) HDel(_ context.Context, key string, fields ...string) error {
	if hash, ok := m.data[key]; ok {
		for _, field := range fields {
			delete(hash, field)
		}
		if len(hash) == 0 {
			delete(m.data, key)
		}
	}

	return nil
}

func (m *mockHash) HExists(_ context.Context, key, field string) (bool, error) {
	if hash, ok := m.data[key]; ok {
		_, exists := hash[field]
		return exists, nil
	}

	return false, nil
}

func (m *mockHash) HKeys(_ context.Context, key string) ([]string, error) {
	if hash, ok := m.data[key]; ok {
		keys := make([]string, 0, len(hash))
		for key := range hash {
			keys = append(keys, key)
		}
		return keys, nil
	}

	return []string{}, nil
}

func (m *mockHash) HVals(_ context.Context, key string) ([][]byte, error) {
	if hash, ok := m.data[key]; ok {
		values := make([][]byte, 0, len(hash))
		for _, value := range hash {
			values = append(values, value)
		}
		return values, nil
	}

	return [][]byte{}, nil
}

func (m *mockHash) HLen(_ context.Context, key string) (int64, error) {
	if hash, ok := m.data[key]; ok {
		return int64(len(hash)), nil
	}

	return 0, nil
}

func (m *mockHash) HIncrBy(_ context.Context, key, field string, _ int64) (int64, error) {
	if _, ok := m.data[key]; !ok {
		m.data[key] = make(map[string][]byte)
	}

	m.data[key][field] = []byte("0")
	return 0, nil
}

type mockKV struct {
	data       map[string][]byte
	expiration map[string]time.Duration
	scanPages  [][]string
}

func newMockKV() *mockKV {
	return &mockKV{
		data:       make(map[string][]byte),
		expiration: make(map[string]time.Duration),
	}
}

func (m *mockKV) Get(_ context.Context, key string) ([]byte, error) {
	if value, ok := m.data[key]; ok {
		return value, nil
	}

	return nil, kvx.ErrNil
}

func (m *mockKV) MGet(_ context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for _, key := range keys {
		if value, ok := m.data[key]; ok {
			result[key] = value
		}
	}

	return result, nil
}

func (m *mockKV) Set(_ context.Context, key string, value []byte, expiration time.Duration) error {
	m.data[key] = value
	if expiration > 0 {
		m.expiration[key] = expiration
	}

	return nil
}

func (m *mockKV) MSet(_ context.Context, values map[string][]byte, expiration time.Duration) error {
	for key, value := range values {
		m.data[key] = value
		if expiration > 0 {
			m.expiration[key] = expiration
		}
	}

	return nil
}

func (m *mockKV) Delete(_ context.Context, key string) error {
	delete(m.data, key)
	delete(m.expiration, key)
	return nil
}

func (m *mockKV) DeleteMulti(_ context.Context, keys []string) error {
	for _, key := range keys {
		delete(m.data, key)
		delete(m.expiration, key)
	}

	return nil
}

func (m *mockKV) Exists(_ context.Context, key string) (bool, error) {
	_, exists := m.data[key]
	return exists, nil
}

func (m *mockKV) ExistsMulti(_ context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		_, exists := m.data[key]
		result[key] = exists
	}

	return result, nil
}

func (m *mockKV) Expire(_ context.Context, key string, expiration time.Duration) error {
	m.expiration[key] = expiration
	return nil
}

func (m *mockKV) TTL(_ context.Context, key string) (time.Duration, error) {
	if ttl, ok := m.expiration[key]; ok {
		return ttl, nil
	}

	return 0, nil
}

func (m *mockKV) Scan(_ context.Context, pattern string, cursor uint64, _ int64) ([]string, uint64, error) {
	if len(m.scanPages) > 0 {
		if cursor > uint64(maxInt) {
			return []string{}, 0, nil
		}

		index := int(cursor)
		if index >= len(m.scanPages) {
			return []string{}, 0, nil
		}

		page := append([]string(nil), m.scanPages[index]...)
		next := uint64(0)
		if index+1 < len(m.scanPages) {
			next = uint64(index + 1)
		}
		return page, next, nil
	}

	matched := make([]string, 0, len(m.data))
	for key := range m.data {
		if matchPattern(key, pattern) {
			matched = append(matched, key)
		}
	}

	return matched, 0, nil
}

func (m *mockKV) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, _, err := m.Scan(ctx, pattern, 0, 0)
	return keys, err
}

func (m *mockKV) Pipeline() kvx.Pipeline {
	return &mockPipeline{kv: m}
}

func (m *mockKV) Close() error {
	return nil
}

type mockJSON struct {
	data map[string][]byte
}

type TestUser struct {
	ID    string `kvx:"id"`
	Name  string `kvx:"name"`
	Email string `kvx:"email,index"`
	Age   int    `kvx:"age,index"`
}

func matchPattern(key, pattern string) bool {
	if len(pattern) > 1 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return key == pattern
}

func fieldNameFromPath(path string) string {
	if len(path) > 2 && path[:2] == "$." {
		return path[2:]
	}

	return path
}
