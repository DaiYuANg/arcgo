// Package valkey provides a Valkey adapter for kvx.
package valkey

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/DaiYuANg/archgo/kvx"
	"github.com/valkey-io/valkey-go"
)

// Adapter implements kvx.Client using valkey-go.
type Adapter struct {
	client valkey.Client
}

// New creates a new Valkey adapter.
func New(opts kvx.ClientOptions) (*Adapter, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: opts.Addrs,
		Password:    opts.Password,
		SelectDB:    opts.DB,
		TLSConfig:   nil, // TODO: support TLS
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}

	return &Adapter{client: client}, nil
}

// NewFromClient creates an adapter from an existing valkey.Client.
func NewFromClient(client valkey.Client) *Adapter {
	return &Adapter{client: client}
}

// Close closes the client connection.
func (a *Adapter) Close() error {
	a.client.Close()
	return nil
}

// ============== KV Interface ==============

// Get retrieves the value for the given key.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Get().Key(key).Build())
	if resp.Error() != nil {
		if valkey.IsValkeyNil(resp.Error()) {
			return nil, kvx.ErrNil
		}
		return nil, resp.Error()
	}
	return resp.AsBytes()
}

// Set sets the value for the given key.
func (a *Adapter) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if expiration > 0 {
		return a.client.Do(ctx, a.client.B().Set().Key(key).Value(valkey.BinaryString(value)).Px(expiration).Build()).Error()
	}
	return a.client.Do(ctx, a.client.B().Set().Key(key).Value(valkey.BinaryString(value)).Build()).Error()
}

// Delete deletes the given key.
func (a *Adapter) Delete(ctx context.Context, key string) error {
	return a.client.Do(ctx, a.client.B().Del().Key(key).Build()).Error()
}

// Exists checks if the key exists.
func (a *Adapter) Exists(ctx context.Context, key string) (bool, error) {
	resp := a.client.Do(ctx, a.client.B().Exists().Key(key).Build())
	if resp.Error() != nil {
		return false, resp.Error()
	}
	n, err := resp.AsInt64()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Expire sets the expiration for the given key.
func (a *Adapter) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return a.client.Do(ctx, a.client.B().Expire().Key(key).Seconds(int64(expiration.Seconds())).Build()).Error()
}

// TTL gets the TTL for the given key.
func (a *Adapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	resp := a.client.Do(ctx, a.client.B().Ttl().Key(key).Build())
	if resp.Error() != nil {
		return 0, resp.Error()
	}
	seconds, err := resp.AsInt64()
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds) * time.Second, nil
}

// ============== Hash Interface ==============

// HGet gets a field from a hash.
func (a *Adapter) HGet(ctx context.Context, key string, field string) ([]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Hget().Key(key).Field(field).Build())
	if resp.Error() != nil {
		if valkey.IsValkeyNil(resp.Error()) {
			return nil, kvx.ErrNil
		}
		return nil, resp.Error()
	}
	return resp.AsBytes()
}

// HSet sets fields in a hash.
func (a *Adapter) HSet(ctx context.Context, key string, values map[string][]byte) error {
	// Build the command with FieldValue chain
	cmd := a.client.B().Hset().Key(key).FieldValue()
	for k, v := range values {
		cmd = cmd.FieldValue(k, valkey.BinaryString(v))
	}
	return a.client.Do(ctx, cmd.Build()).Error()
}

// HGetAll gets all fields and values from a hash.
func (a *Adapter) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Hgetall().Key(key).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	m, err := resp.AsStrMap()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(m))
	for k, v := range m {
		result[k] = []byte(v)
	}
	return result, nil
}

// HDel deletes fields from a hash.
func (a *Adapter) HDel(ctx context.Context, key string, fields ...string) error {
	return a.client.Do(ctx, a.client.B().Hdel().Key(key).Field(fields...).Build()).Error()
}

// HExists checks if a field exists in a hash.
func (a *Adapter) HExists(ctx context.Context, key string, field string) (bool, error) {
	resp := a.client.Do(ctx, a.client.B().Hexists().Key(key).Field(field).Build())
	if resp.Error() != nil {
		return false, resp.Error()
	}
	return resp.AsBool()
}

// HKeys gets all field names in a hash.
func (a *Adapter) HKeys(ctx context.Context, key string) ([]string, error) {
	resp := a.client.Do(ctx, a.client.B().Hkeys().Key(key).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}
	return resp.AsStrSlice()
}

// HVals gets all values in a hash.
func (a *Adapter) HVals(ctx context.Context, key string) ([][]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Hvals().Key(key).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}
	strs, err := resp.AsStrSlice()
	if err != nil {
		return nil, err
	}
	result := make([][]byte, len(strs))
	for i, v := range strs {
		result[i] = []byte(v)
	}
	return result, nil
}

// HLen gets the number of fields in a hash.
func (a *Adapter) HLen(ctx context.Context, key string) (int64, error) {
	resp := a.client.Do(ctx, a.client.B().Hlen().Key(key).Build())
	if resp.Error() != nil {
		return 0, resp.Error()
	}
	return resp.AsInt64()
}

// ============== PubSub Interface ==============

// Publish publishes a message to a channel.
func (a *Adapter) Publish(ctx context.Context, channel string, message []byte) error {
	return a.client.Do(ctx, a.client.B().Publish().Channel(channel).Message(valkey.BinaryString(message)).Build()).Error()
}

// Subscribe subscribes to a channel.
func (a *Adapter) Subscribe(ctx context.Context, channel string) (kvx.Subscription, error) {
	sub := &valkeySubscription{
		client:  a.client,
		channel: channel,
		ch:      make(chan []byte, 100),
		ctx:     ctx,
	}

	// Start receiving messages
	go func() {
		defer close(sub.ch)
		err := a.client.Receive(ctx, a.client.B().Subscribe().Channel(channel).Build(), func(msg valkey.PubSubMessage) {
			sub.ch <- []byte(msg.Message)
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			// Connection closed or error
		}
	}()

	return sub, nil
}

// PSubscribe subscribes to channels matching a pattern.
func (a *Adapter) PSubscribe(ctx context.Context, pattern string) (kvx.Subscription, error) {
	sub := &valkeySubscription{
		client:  a.client,
		pattern: pattern,
		ch:      make(chan []byte, 100),
		ctx:     ctx,
	}

	// Start receiving messages
	go func() {
		defer close(sub.ch)
		err := a.client.Receive(ctx, a.client.B().Psubscribe().Pattern(pattern).Build(), func(msg valkey.PubSubMessage) {
			sub.ch <- []byte(msg.Message)
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			// Connection closed or error
		}
	}()

	return sub, nil
}

type valkeySubscription struct {
	client  valkey.Client
	channel string
	pattern string
	ch      chan []byte
	ctx     context.Context
	mu      sync.Mutex
	closed  bool
}

func (s *valkeySubscription) Channel() <-chan []byte {
	return s.ch
}

func (s *valkeySubscription) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	// The channel will be closed when Receive returns
	return nil
}

// ============== Stream Interface ==============

// XAdd adds an entry to a stream.
func (a *Adapter) XAdd(ctx context.Context, key string, id string, values map[string][]byte) (string, error) {
	// Build the command with FieldValue chain
	cmd := a.client.B().Xadd().Key(key).Id(id).FieldValue()
	for k, v := range values {
		cmd = cmd.FieldValue(k, valkey.BinaryString(v))
	}

	resp := a.client.Do(ctx, cmd.Build())
	if resp.Error() != nil {
		return "", resp.Error()
	}
	return resp.ToString()
}

// XRead reads entries from a stream.
func (a *Adapter) XRead(ctx context.Context, key string, start string, count int64) ([]kvx.StreamEntry, error) {
	var cmd valkey.Completed
	if count > 0 {
		cmd = a.client.B().Xread().Count(count).Block(0).Streams().Key(key).Id(start).Build()
	} else {
		cmd = a.client.B().Xread().Block(0).Streams().Key(key).Id(start).Build()
	}

	resp := a.client.Do(ctx, cmd)
	if resp.Error() != nil {
		if valkey.IsValkeyNil(resp.Error()) {
			return nil, nil
		}
		return nil, resp.Error()
	}

	// Parse XREAD response using AsXRead
	xreadResult, err := resp.AsXRead()
	if err != nil {
		return nil, err
	}

	entries := make([]kvx.StreamEntry, 0)
	for _, streamEntries := range xreadResult {
		for _, entry := range streamEntries {
			values := make(map[string][]byte)
			for f, v := range entry.FieldValues {
				values[f] = []byte(v)
			}
			entries = append(entries, kvx.StreamEntry{
				ID:     entry.ID,
				Values: values,
			})
		}
	}

	return entries, nil
}

// XRange reads entries in a range.
func (a *Adapter) XRange(ctx context.Context, key string, start, stop string) ([]kvx.StreamEntry, error) {
	resp := a.client.Do(ctx, a.client.B().Xrange().Key(key).Start(start).End(stop).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	// Parse XRANGE response using AsXRange
	xrangeEntries, err := resp.AsXRange()
	if err != nil {
		return nil, err
	}

	entries := make([]kvx.StreamEntry, len(xrangeEntries))
	for i, entry := range xrangeEntries {
		values := make(map[string][]byte)
		for f, v := range entry.FieldValues {
			values[f] = []byte(v)
		}
		entries[i] = kvx.StreamEntry{
			ID:     entry.ID,
			Values: values,
		}
	}

	return entries, nil
}

// XLen gets the number of entries in a stream.
func (a *Adapter) XLen(ctx context.Context, key string) (int64, error) {
	resp := a.client.Do(ctx, a.client.B().Xlen().Key(key).Build())
	if resp.Error() != nil {
		return 0, resp.Error()
	}
	return resp.AsInt64()
}

// XTrim trims the stream to approximately maxLen entries.
func (a *Adapter) XTrim(ctx context.Context, key string, maxLen int64) error {
	return a.client.Do(ctx, a.client.B().Xtrim().Key(key).Maxlen().Threshold(strconv.FormatInt(maxLen, 10)).Build()).Error()
}

// ============== Script Interface ==============

// Load loads a script into the script cache.
func (a *Adapter) Load(ctx context.Context, script string) (string, error) {
	resp := a.client.Do(ctx, a.client.B().ScriptLoad().Script(script).Build())
	if resp.Error() != nil {
		return "", resp.Error()
	}
	return resp.ToString()
}

// Eval executes a script.
func (a *Adapter) Eval(ctx context.Context, script string, keys []string, args [][]byte) ([]byte, error) {
	// Build eval command
	argStrs := make([]string, len(args))
	for i, arg := range args {
		argStrs[i] = valkey.BinaryString(arg)
	}

	cmd := a.client.B().Eval().Script(script).Numkeys(int64(len(keys))).Key(keys...).Arg(argStrs...)

	resp := a.client.Do(ctx, cmd.Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	return resp.AsBytes()
}

// EvalSHA executes a cached script by SHA.
func (a *Adapter) EvalSHA(ctx context.Context, sha string, keys []string, args [][]byte) ([]byte, error) {
	argStrs := make([]string, len(args))
	for i, arg := range args {
		argStrs[i] = valkey.BinaryString(arg)
	}

	cmd := a.client.B().Evalsha().Sha1(sha).Numkeys(int64(len(keys))).Key(keys...).Arg(argStrs...)

	resp := a.client.Do(ctx, cmd.Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	return resp.AsBytes()
}

// ============== JSON Interface ==============

// JSONSet sets a JSON value at key.
func (a *Adapter) JSONSet(ctx context.Context, key string, path string, value []byte, expiration time.Duration) error {
	err := a.client.Do(ctx, a.client.B().JsonSet().Key(key).Path(path).Value(valkey.BinaryString(value)).Build()).Error()
	if err != nil {
		return err
	}

	if expiration > 0 {
		return a.client.Do(ctx, a.client.B().Expire().Key(key).Seconds(int64(expiration.Seconds())).Build()).Error()
	}
	return nil
}

// JSONGet gets a JSON value at key.
func (a *Adapter) JSONGet(ctx context.Context, key string, path string) ([]byte, error) {
	resp := a.client.Do(ctx, a.client.B().JsonGet().Key(key).Path(path).Build())
	if resp.Error() != nil {
		if valkey.IsValkeyNil(resp.Error()) {
			return nil, kvx.ErrNil
		}
		return nil, resp.Error()
	}
	return resp.AsBytes()
}

// JSONSetField sets a field in a JSON document.
func (a *Adapter) JSONSetField(ctx context.Context, key string, path string, value []byte) error {
	return a.client.Do(ctx, a.client.B().JsonSet().Key(key).Path(path).Value(valkey.BinaryString(value)).Build()).Error()
}

// JSONGetField gets a field from a JSON document.
func (a *Adapter) JSONGetField(ctx context.Context, key string, path string) ([]byte, error) {
	return a.JSONGet(ctx, key, path)
}

// JSONDelete deletes a JSON value or field.
func (a *Adapter) JSONDelete(ctx context.Context, key string, path string) error {
	return a.client.Do(ctx, a.client.B().JsonDel().Key(key).Path(path).Build()).Error()
}

// ============== Search Interface ==============

// CreateIndex creates a secondary index.
func (a *Adapter) CreateIndex(ctx context.Context, indexName string, prefix string, schema []kvx.SchemaField) error {
	args := []string{indexName, "ON", "HASH", "PREFIX", "1", prefix, "SCHEMA"}

	for _, f := range schema {
		args = append(args, f.Name, string(f.Type))
		if f.Sortable {
			args = append(args, "SORTABLE")
		}
	}

	return a.client.Do(ctx, a.client.B().Arbitrary("FT.CREATE").Args(args...).Build()).Error()
}

// DropIndex drops a secondary index.
func (a *Adapter) DropIndex(ctx context.Context, indexName string) error {
	return a.client.Do(ctx, a.client.B().Arbitrary("FT.DROPINDEX").Args(indexName).Build()).Error()
}

// Search performs a search query.
func (a *Adapter) Search(ctx context.Context, indexName string, query string, limit int) ([]string, error) {
	resp := a.client.Do(ctx, a.client.B().Arbitrary("FT.SEARCH").Args(indexName, query, "LIMIT", "0", fmt.Sprintf("%d", limit)).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	// Use AsFtSearch to parse the response
	total, docs, err := resp.AsFtSearch()
	if err != nil {
		return nil, err
	}
	_ = total // We don't need the total count here

	keys := make([]string, len(docs))
	for i, doc := range docs {
		keys[i] = doc.Key
	}
	return keys, nil
}

// ============== Pipeline Interface ==============

// Pipeline creates a new pipeline.
func (a *Adapter) Pipeline() kvx.Pipeline {
	return &valkeyPipeline{
		client: a.client,
	}
}

type valkeyPipeline struct {
	client valkey.Client
	cmds   []valkey.Completed
}

// Enqueue adds a command to the pipeline.
func (p *valkeyPipeline) Enqueue(command string, args ...[]byte) {
	argStrs := make([]string, len(args))
	for i, v := range args {
		argStrs[i] = valkey.BinaryString(v)
	}
	cmd := p.client.B().Arbitrary(command).Args(argStrs...).Build()
	p.cmds = append(p.cmds, cmd)
}

// Exec executes all queued commands.
func (p *valkeyPipeline) Exec(ctx context.Context) ([][]byte, error) {
	if len(p.cmds) == 0 {
		return nil, nil
	}

	// Use DoMulti for pipeline execution
	resps := p.client.DoMulti(ctx, p.cmds...)

	results := make([][]byte, len(resps))
	for i, resp := range resps {
		if resp.Error() != nil && !valkey.IsValkeyNil(resp.Error()) {
			results[i] = nil
			continue
		}
		b, _ := resp.AsBytes()
		results[i] = b
	}
	return results, nil
}

// Close closes the pipeline.
func (p *valkeyPipeline) Close() error {
	// No explicit close needed
	return nil
}

// ============== Lock Interface ==============

// Acquire tries to acquire a lock.
func (a *Adapter) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	// Use SET NX for simple distributed lock
	resp := a.client.Do(ctx, a.client.B().Set().Key(key).Value("1").Nx().Px(ttl).Build())
	if resp.Error() != nil {
		if valkey.IsValkeyNil(resp.Error()) {
			return false, nil
		}
		return false, resp.Error()
	}
	return true, nil
}

// Release releases a lock.
func (a *Adapter) Release(ctx context.Context, key string) error {
	return a.client.Do(ctx, a.client.B().Del().Key(key).Build()).Error()
}

// Extend extends the lock TTL.
func (a *Adapter) Extend(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	resp := a.client.Do(ctx, a.client.B().Expire().Key(key).Seconds(int64(ttl.Seconds())).Build())
	if resp.Error() != nil {
		return false, resp.Error()
	}
	return resp.AsBool()
}
