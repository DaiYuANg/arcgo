// Package redis provides a Redis adapter for kvx.
package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/DaiYuANg/archgo/kvx"
	"github.com/redis/go-redis/v9"
)

// Adapter implements kvx.Client using go-redis.
type Adapter struct {
	client *redis.Client
}

// New creates a new Redis adapter.
func New(opts kvx.ClientOptions) (*Adapter, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:            opts.Addrs[0],
		Password:        opts.Password,
		DB:              opts.DB,
		TLSConfig:       nil, // TODO: support TLS
		PoolSize:        opts.PoolSize,
		MinIdleConns:    opts.MinIdleConns,
		ConnMaxLifetime: opts.ConnMaxLifetime,
		ConnMaxIdleTime: opts.ConnMaxIdleTime,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Adapter{client: rdb}, nil
}

// NewFromClient creates an adapter from an existing redis.Client.
func NewFromClient(client *redis.Client) *Adapter {
	return &Adapter{client: client}
}

// Close closes the client connection.
func (a *Adapter) Close() error {
	return a.client.Close()
}

// ============== KV Interface ==============

// Get retrieves the value for the given key.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := a.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, kvx.ErrNil
		}
		return nil, err
	}
	return []byte(val), nil
}

// Set sets the value for the given key.
func (a *Adapter) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	return a.client.Set(ctx, key, value, expiration).Err()
}

// Delete deletes the given key.
func (a *Adapter) Delete(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}

// Exists checks if the key exists.
func (a *Adapter) Exists(ctx context.Context, key string) (bool, error) {
	n, err := a.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Expire sets the expiration for the given key.
func (a *Adapter) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return a.client.Expire(ctx, key, expiration).Err()
}

// TTL gets the TTL for the given key.
func (a *Adapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	return a.client.TTL(ctx, key).Result()
}

// ============== Hash Interface ==============

// HGet gets a field from a hash.
func (a *Adapter) HGet(ctx context.Context, key string, field string) ([]byte, error) {
	val, err := a.client.HGet(ctx, key, field).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, kvx.ErrNil
		}
		return nil, err
	}
	return []byte(val), nil
}

// HSet sets fields in a hash.
func (a *Adapter) HSet(ctx context.Context, key string, values map[string][]byte) error {
	// Convert map[string][]byte to map[string]interface{}
	ifaceValues := make(map[string]interface{}, len(values))
	for k, v := range values {
		ifaceValues[k] = v
	}
	return a.client.HSet(ctx, key, ifaceValues).Err()
}

// HGetAll gets all fields and values from a hash.
func (a *Adapter) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	val, err := a.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(val))
	for k, v := range val {
		result[k] = []byte(v)
	}
	return result, nil
}

// HDel deletes fields from a hash.
func (a *Adapter) HDel(ctx context.Context, key string, fields ...string) error {
	return a.client.HDel(ctx, key, fields...).Err()
}

// HExists checks if a field exists in a hash.
func (a *Adapter) HExists(ctx context.Context, key string, field string) (bool, error) {
	return a.client.HExists(ctx, key, field).Result()
}

// HKeys gets all field names in a hash.
func (a *Adapter) HKeys(ctx context.Context, key string) ([]string, error) {
	return a.client.HKeys(ctx, key).Result()
}

// HVals gets all values in a hash.
func (a *Adapter) HVals(ctx context.Context, key string) ([][]byte, error) {
	vals, err := a.client.HVals(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	result := make([][]byte, len(vals))
	for i, v := range vals {
		result[i] = []byte(v)
	}
	return result, nil
}

// HLen gets the number of fields in a hash.
func (a *Adapter) HLen(ctx context.Context, key string) (int64, error) {
	return a.client.HLen(ctx, key).Result()
}

// ============== PubSub Interface ==============

// Publish publishes a message to a channel.
func (a *Adapter) Publish(ctx context.Context, channel string, message []byte) error {
	return a.client.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to a channel.
func (a *Adapter) Subscribe(ctx context.Context, channel string) (kvx.Subscription, error) {
	pubsub := a.client.Subscribe(ctx, channel)
	// Verify subscription
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, err
	}
	return &redisSubscription{pubsub: pubsub}, nil
}

// PSubscribe subscribes to channels matching a pattern.
func (a *Adapter) PSubscribe(ctx context.Context, pattern string) (kvx.Subscription, error) {
	pubsub := a.client.PSubscribe(ctx, pattern)
	// Verify subscription
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, err
	}
	return &redisSubscription{pubsub: pubsub}, nil
}

type redisSubscription struct {
	pubsub *redis.PubSub
	once   sync.Once
	ch     chan []byte
}

func (s *redisSubscription) Channel() <-chan []byte {
	s.once.Do(func() {
		s.ch = make(chan []byte, 100)
		go func() {
			defer close(s.ch)
			ch := s.pubsub.Channel()
			for msg := range ch {
				s.ch <- []byte(msg.Payload)
			}
		}()
	})
	return s.ch
}

func (s *redisSubscription) Close() error {
	return s.pubsub.Close()
}

// ============== Stream Interface ==============

// XAdd adds an entry to a stream.
func (a *Adapter) XAdd(ctx context.Context, key string, id string, values map[string][]byte) (string, error) {
	// Convert map[string][]byte to map[string]interface{}
	ifaceValues := make(map[string]interface{}, len(values))
	for k, v := range values {
		ifaceValues[k] = v
	}

	args := &redis.XAddArgs{
		Stream: key,
		Values: ifaceValues,
	}
	if id != "*" {
		args.ID = id
	}

	return a.client.XAdd(ctx, args).Result()
}

// XRead reads entries from a stream.
func (a *Adapter) XRead(ctx context.Context, key string, start string, count int64) ([]kvx.StreamEntry, error) {
	streams := []string{key, start}

	result, err := a.client.XRead(ctx, &redis.XReadArgs{
		Streams: streams,
		Count:   count,
		Block:   0,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	entries := make([]kvx.StreamEntry, len(result[0].Messages))
	for i, msg := range result[0].Messages {
		entries[i] = kvx.StreamEntry{
			ID:     msg.ID,
			Values: convertInterfaceMapToBytes(msg.Values),
		}
	}
	return entries, nil
}

// XRange reads entries in a range.
func (a *Adapter) XRange(ctx context.Context, key string, start, stop string) ([]kvx.StreamEntry, error) {
	result, err := a.client.XRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]kvx.StreamEntry, len(result))
	for i, msg := range result {
		entries[i] = kvx.StreamEntry{
			ID:     msg.ID,
			Values: convertInterfaceMapToBytes(msg.Values),
		}
	}
	return entries, nil
}

// XLen gets the number of entries in a stream.
func (a *Adapter) XLen(ctx context.Context, key string) (int64, error) {
	return a.client.XLen(ctx, key).Result()
}

// XTrim trims the stream to approximately maxLen entries.
func (a *Adapter) XTrim(ctx context.Context, key string, maxLen int64) error {
	return a.client.XTrimMaxLen(ctx, key, maxLen).Err()
}

// ============== Script Interface ==============

// Load loads a script into the script cache.
func (a *Adapter) Load(ctx context.Context, script string) (string, error) {
	return a.client.ScriptLoad(ctx, script).Result()
}

// Eval executes a script.
func (a *Adapter) Eval(ctx context.Context, script string, keys []string, args [][]byte) ([]byte, error) {
	ifaceArgs := make([]interface{}, len(args))
	for i, v := range args {
		ifaceArgs[i] = v
	}

	val, err := a.client.Eval(ctx, script, keys, ifaceArgs...).Result()
	if err != nil {
		return nil, err
	}

	return valueToBytes(val)
}

// EvalSHA executes a cached script by SHA.
func (a *Adapter) EvalSHA(ctx context.Context, sha string, keys []string, args [][]byte) ([]byte, error) {
	ifaceArgs := make([]interface{}, len(args))
	for i, v := range args {
		ifaceArgs[i] = v
	}

	val, err := a.client.EvalSha(ctx, sha, keys, ifaceArgs...).Result()
	if err != nil {
		return nil, err
	}

	return valueToBytes(val)
}

// ============== JSON Interface ==============

// JSONSet sets a JSON value at key.
func (a *Adapter) JSONSet(ctx context.Context, key string, path string, value []byte, expiration time.Duration) error {
	// Use JSON.SET command via Do
	err := a.client.Do(ctx, "JSON.SET", key, path, value).Err()
	if err != nil {
		return err
	}

	if expiration > 0 {
		return a.client.Expire(ctx, key, expiration).Err()
	}
	return nil
}

// JSONGet gets a JSON value at key.
func (a *Adapter) JSONGet(ctx context.Context, key string, path string) ([]byte, error) {
	val, err := a.client.Do(ctx, "JSON.GET", key, path).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, kvx.ErrNil
		}
		return nil, err
	}

	return valueToBytes(val)
}

// JSONSetField sets a field in a JSON document.
func (a *Adapter) JSONSetField(ctx context.Context, key string, path string, value []byte) error {
	return a.client.Do(ctx, "JSON.SET", key, path, value).Err()
}

// JSONGetField gets a field from a JSON document.
func (a *Adapter) JSONGetField(ctx context.Context, key string, path string) ([]byte, error) {
	return a.JSONGet(ctx, key, path)
}

// JSONDelete deletes a JSON value or field.
func (a *Adapter) JSONDelete(ctx context.Context, key string, path string) error {
	return a.client.Do(ctx, "JSON.DEL", key, path).Err()
}

// ============== Search Interface ==============

// CreateIndex creates a secondary index.
func (a *Adapter) CreateIndex(ctx context.Context, indexName string, prefix string, schema []kvx.SchemaField) error {
	args := make([]interface{}, 0)
	args = append(args, indexName, "ON", "HASH", "PREFIX", 1, prefix, "SCHEMA")

	for _, f := range schema {
		args = append(args, f.Name, string(f.Type))
		if f.Sortable {
			args = append(args, "SORTABLE")
		}
	}

	return a.client.Do(ctx, args...).Err()
}

// DropIndex drops a secondary index.
func (a *Adapter) DropIndex(ctx context.Context, indexName string) error {
	return a.client.Do(ctx, "FT.DROPINDEX", indexName).Err()
}

// Search performs a search query.
func (a *Adapter) Search(ctx context.Context, indexName string, query string, limit int) ([]string, error) {
	val, err := a.client.Do(ctx, "FT.SEARCH", indexName, query, "LIMIT", 0, limit).Result()
	if err != nil {
		return nil, err
	}

	// Parse FT.SEARCH response
	// Response format: [total, key1, [field1, value1, ...], key2, ...]
	return parseFTSearchResponse(val)
}

// ============== Pipeline Interface ==============

// Pipeline creates a new pipeline.
func (a *Adapter) Pipeline() kvx.Pipeline {
	return &redisPipeline{
		pipe: a.client.Pipeline(),
	}
}

type redisPipeline struct {
	pipe redis.Pipeliner
}

// Enqueue adds a command to the pipeline.
func (p *redisPipeline) Enqueue(command string, args ...[]byte) {
	// Convert args to interface{}
	ifaceArgs := make([]interface{}, len(args)+1)
	ifaceArgs[0] = command
	for i, v := range args {
		ifaceArgs[i+1] = v
	}
	p.pipe.Do(context.Background(), ifaceArgs...)
}

// Exec executes all queued commands.
func (p *redisPipeline) Exec(ctx context.Context) ([][]byte, error) {
	cmders, err := p.pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	results := make([][]byte, len(cmders))
	for i, cmd := range cmders {
		val, err := cmd.(*redis.Cmd).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			results[i] = nil
			continue
		}
		results[i], _ = valueToBytes(val)
	}
	return results, nil
}

// Close closes the pipeline.
func (p *redisPipeline) Close() error {
	// Pipeline doesn't need explicit close in go-redis
	return nil
}

// ============== Lock Interface ==============

// Acquire tries to acquire a lock.
func (a *Adapter) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	// Use SET NX for simple distributed lock
	ok, err := a.client.SetNX(ctx, key, "1", ttl).Result()
	return ok, err
}

// Release releases a lock.
func (a *Adapter) Release(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}

// Extend extends the lock TTL.
func (a *Adapter) Extend(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	// Use PEXPIRE to extend the lock
	ok, err := a.client.Expire(ctx, key, ttl).Result()
	return ok, err
}

// ============== Helper Functions ==============

func convertInterfaceMapToBytes(m map[string]interface{}) map[string][]byte {
	result := make(map[string][]byte, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case []byte:
			result[k] = val
		case string:
			result[k] = []byte(val)
		default:
			result[k] = []byte(fmt.Sprintf("%v", val))
		}
	}
	return result
}

func valueToBytes(val interface{}) ([]byte, error) {
	switch v := val.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	case nil:
		return nil, nil
	default:
		return []byte(fmt.Sprintf("%v", v)), nil
	}
}

func parseFTSearchResponse(val interface{}) ([]string, error) {
	arr, ok := val.([]interface{})
	if !ok {
		return nil, nil
	}

	if len(arr) < 1 {
		return nil, nil
	}

	// Extract keys from the response
	var keys []string
	for i := 1; i < len(arr); i += 2 {
		if key, ok := arr[i].(string); ok {
			keys = append(keys, key)
		}
	}
	return keys, nil
}
