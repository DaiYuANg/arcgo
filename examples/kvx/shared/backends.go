package shared

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
)

type User struct {
	ID    string `kvx:"id"`
	Name  string `kvx:"name"`
	Email string `kvx:"email,index=email"`
}

type HashBackend struct {
	hashes     map[string]map[string][]byte
	keys       map[string][]byte
	expiration map[string]time.Duration
}

func NewHashBackend() *HashBackend {
	return &HashBackend{
		hashes:     make(map[string]map[string][]byte),
		keys:       make(map[string][]byte),
		expiration: make(map[string]time.Duration),
	}
}

func (b *HashBackend) Get(ctx context.Context, key string) ([]byte, error) {
	value, ok := b.keys[key]
	if !ok {
		return nil, kvx.ErrNil
	}
	return value, nil
}

func (b *HashBackend) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(keys))
	for _, key := range keys {
		if value, ok := b.keys[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (b *HashBackend) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	b.keys[key] = value
	if expiration > 0 {
		b.expiration[key] = expiration
	}
	return nil
}

func (b *HashBackend) MSet(ctx context.Context, values map[string][]byte, expiration time.Duration) error {
	for key, value := range values {
		if err := b.Set(ctx, key, value, expiration); err != nil {
			return err
		}
	}
	return nil
}

func (b *HashBackend) Delete(ctx context.Context, key string) error {
	delete(b.keys, key)
	delete(b.hashes, key)
	delete(b.expiration, key)
	return nil
}

func (b *HashBackend) DeleteMulti(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := b.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (b *HashBackend) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := b.keys[key]
	return ok, nil
}

func (b *HashBackend) ExistsMulti(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))
	for _, key := range keys {
		_, ok := b.keys[key]
		result[key] = ok
	}
	return result, nil
}

func (b *HashBackend) Expire(ctx context.Context, key string, expiration time.Duration) error {
	b.expiration[key] = expiration
	return nil
}

func (b *HashBackend) TTL(ctx context.Context, key string) (time.Duration, error) {
	return b.expiration[key], nil
}

func (b *HashBackend) Scan(ctx context.Context, pattern string, cursor uint64, count int64) ([]string, uint64, error) {
	keys := make([]string, 0, len(b.keys))
	for key := range b.keys {
		if matchesPattern(key, pattern) {
			keys = append(keys, key)
		}
	}
	return keys, 0, nil
}

func (b *HashBackend) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, _, err := b.Scan(ctx, pattern, 0, 0)
	return keys, err
}

func (b *HashBackend) HGet(ctx context.Context, key string, field string) ([]byte, error) {
	hash, ok := b.hashes[key]
	if !ok {
		return nil, kvx.ErrNil
	}
	value, ok := hash[field]
	if !ok {
		return nil, kvx.ErrNil
	}
	return value, nil
}

func (b *HashBackend) HMGet(ctx context.Context, key string, fields []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(fields))
	for _, field := range fields {
		value, err := b.HGet(ctx, key, field)
		if err == nil {
			result[field] = value
		}
	}
	return result, nil
}

func (b *HashBackend) HSet(ctx context.Context, key string, values map[string][]byte) error {
	if _, ok := b.hashes[key]; !ok {
		b.hashes[key] = make(map[string][]byte)
	}
	for field, value := range values {
		b.hashes[key][field] = value
	}
	b.keys[key] = []byte("1")
	return nil
}

func (b *HashBackend) HMSet(ctx context.Context, key string, values map[string][]byte) error {
	return b.HSet(ctx, key, values)
}

func (b *HashBackend) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	hash, ok := b.hashes[key]
	if !ok {
		return map[string][]byte{}, nil
	}
	result := make(map[string][]byte, len(hash))
	for field, value := range hash {
		result[field] = value
	}
	return result, nil
}

func (b *HashBackend) HDel(ctx context.Context, key string, fields ...string) error {
	hash, ok := b.hashes[key]
	if !ok {
		return nil
	}
	for _, field := range fields {
		delete(hash, field)
	}
	if len(hash) == 0 {
		delete(b.hashes, key)
		delete(b.keys, key)
	}
	return nil
}

func (b *HashBackend) HExists(ctx context.Context, key string, field string) (bool, error) {
	hash, ok := b.hashes[key]
	if !ok {
		return false, nil
	}
	_, ok = hash[field]
	return ok, nil
}

func (b *HashBackend) HKeys(ctx context.Context, key string) ([]string, error) {
	hash, ok := b.hashes[key]
	if !ok {
		return nil, nil
	}
	keys := make([]string, 0, len(hash))
	for field := range hash {
		keys = append(keys, field)
	}
	return keys, nil
}

func (b *HashBackend) HVals(ctx context.Context, key string) ([][]byte, error) {
	hash, ok := b.hashes[key]
	if !ok {
		return nil, nil
	}
	values := make([][]byte, 0, len(hash))
	for _, value := range hash {
		values = append(values, value)
	}
	return values, nil
}

func (b *HashBackend) HLen(ctx context.Context, key string) (int64, error) {
	return int64(len(b.hashes[key])), nil
}

func (b *HashBackend) HIncrBy(ctx context.Context, key string, field string, increment int64) (int64, error) {
	current, _ := b.HGet(ctx, key, field)
	value, _ := strconv.ParseInt(string(current), 10, 64)
	value += increment
	if err := b.HSet(ctx, key, map[string][]byte{field: []byte(strconv.FormatInt(value, 10))}); err != nil {
		return 0, err
	}
	return value, nil
}

type JSONBackend struct {
	data       map[string][]byte
	expiration map[string]time.Duration
}

func NewJSONBackend() *JSONBackend {
	return &JSONBackend{
		data:       make(map[string][]byte),
		expiration: make(map[string]time.Duration),
	}
}

func (b *JSONBackend) Get(ctx context.Context, key string) ([]byte, error) {
	value, ok := b.data[key]
	if !ok {
		return nil, kvx.ErrNil
	}
	return value, nil
}

func (b *JSONBackend) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(keys))
	for _, key := range keys {
		if value, ok := b.data[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (b *JSONBackend) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	b.data[key] = value
	if expiration > 0 {
		b.expiration[key] = expiration
	}
	return nil
}

func (b *JSONBackend) MSet(ctx context.Context, values map[string][]byte, expiration time.Duration) error {
	for key, value := range values {
		if err := b.Set(ctx, key, value, expiration); err != nil {
			return err
		}
	}
	return nil
}

func (b *JSONBackend) Delete(ctx context.Context, key string) error {
	delete(b.data, key)
	delete(b.expiration, key)
	return nil
}

func (b *JSONBackend) DeleteMulti(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := b.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (b *JSONBackend) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := b.data[key]
	return ok, nil
}

func (b *JSONBackend) ExistsMulti(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))
	for _, key := range keys {
		_, ok := b.data[key]
		result[key] = ok
	}
	return result, nil
}

func (b *JSONBackend) Expire(ctx context.Context, key string, expiration time.Duration) error {
	b.expiration[key] = expiration
	return nil
}

func (b *JSONBackend) TTL(ctx context.Context, key string) (time.Duration, error) {
	return b.expiration[key], nil
}

func (b *JSONBackend) Scan(ctx context.Context, pattern string, cursor uint64, count int64) ([]string, uint64, error) {
	keys := make([]string, 0, len(b.data))
	for key := range b.data {
		if matchesPattern(key, pattern) {
			keys = append(keys, key)
		}
	}
	return keys, 0, nil
}

func (b *JSONBackend) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, _, err := b.Scan(ctx, pattern, 0, 0)
	return keys, err
}

func (b *JSONBackend) JSONSet(ctx context.Context, key string, path string, value []byte, expiration time.Duration) error {
	return b.Set(ctx, key, value, expiration)
}

func (b *JSONBackend) JSONGet(ctx context.Context, key string, path string) ([]byte, error) {
	value, ok := b.data[key]
	if !ok {
		return nil, nil
	}
	return value, nil
}

func (b *JSONBackend) JSONSetField(ctx context.Context, key string, path string, value []byte) error {
	current, ok := b.data[key]
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

	document[strings.TrimPrefix(path, "$.")] = fieldValue

	encoded, err := json.Marshal(document)
	if err != nil {
		return err
	}

	b.data[key] = encoded
	return nil
}

func (b *JSONBackend) JSONGetField(ctx context.Context, key string, path string) ([]byte, error) {
	current, ok := b.data[key]
	if !ok {
		return nil, nil
	}

	var document map[string]json.RawMessage
	if err := json.Unmarshal(current, &document); err != nil {
		return nil, err
	}

	return document[strings.TrimPrefix(path, "$.")], nil
}

func (b *JSONBackend) JSONDelete(ctx context.Context, key string, path string) error {
	return b.Delete(ctx, key)
}

func matchesPattern(key string, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(key, strings.TrimSuffix(pattern, "*"))
	}
	return key == pattern
}
