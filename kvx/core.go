package kvx

import (
	"context"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
)

// KV is the base key-value interface.
type KV interface {
	// Get retrieves the value for the given key.
	Get(ctx context.Context, key string) ([]byte, error)
	// MGet retrieves multiple values for the given keys.
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	// Set sets the value for the given key.
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	// MSet sets multiple key-value pairs.
	MSet(ctx context.Context, values map[string][]byte, expiration time.Duration) error
	// Delete deletes the given key.
	Delete(ctx context.Context, key string) error
	// DeleteMulti deletes multiple keys.
	DeleteMulti(ctx context.Context, keys []string) error
	// Exists checks if the key exists.
	Exists(ctx context.Context, key string) (bool, error)
	// ExistsMulti checks if multiple keys exist.
	ExistsMulti(ctx context.Context, keys []string) (map[string]bool, error)
	// Expire sets the expiration for the given key.
	Expire(ctx context.Context, key string, expiration time.Duration) error
	// TTL gets the TTL for the given key.
	TTL(ctx context.Context, key string) (time.Duration, error)
	// Scan iterates over keys matching the pattern.
	Scan(ctx context.Context, pattern string, cursor uint64, count int64) (collectionx.List[string], uint64, error)
	// Keys returns all keys matching the pattern (use with caution on large datasets).
	Keys(ctx context.Context, pattern string) (collectionx.List[string], error)
}

// Hash represents a hash (field-value map) operation.
type Hash interface {
	// HGet gets a field from a hash.
	HGet(ctx context.Context, key string, field string) ([]byte, error)
	// HMGet gets multiple fields from a hash.
	HMGet(ctx context.Context, key string, fields []string) (map[string][]byte, error)
	// HSet sets fields in a hash.
	HSet(ctx context.Context, key string, values map[string][]byte) error
	// HMSet sets multiple fields in a hash (atomic operation).
	HMSet(ctx context.Context, key string, values map[string][]byte) error
	// HGetAll gets all fields and values from a hash.
	HGetAll(ctx context.Context, key string) (map[string][]byte, error)
	// HDel deletes fields from a hash.
	HDel(ctx context.Context, key string, fields ...string) error
	// HExists checks if a field exists in a hash.
	HExists(ctx context.Context, key string, field string) (bool, error)
	// HKeys gets all field names in a hash.
	HKeys(ctx context.Context, key string) (collectionx.List[string], error)
	// HVals gets all values in a hash.
	HVals(ctx context.Context, key string) (collectionx.List[[]byte], error)
	// HLen gets the number of fields in a hash.
	HLen(ctx context.Context, key string) (int64, error)
	// HIncrBy increments a field by the given value.
	HIncrBy(ctx context.Context, key string, field string, increment int64) (int64, error)
}

// PubSub represents pub/sub operations.
type PubSub interface {
	// Publish publishes a message to a channel.
	Publish(ctx context.Context, channel string, message []byte) error
	// Subscribe subscribes to a channel.
	Subscribe(ctx context.Context, channel string) (Subscription, error)
	// PSubscribe subscribes to channels matching a pattern.
	PSubscribe(ctx context.Context, pattern string) (Subscription, error)
}

// Subscription represents a pub/sub subscription.
type Subscription interface {
	// Channel returns the channel for receiving messages.
	Channel() <-chan []byte
	// Close closes the subscription.
	Close() error
}

// Stream represents stream operations.
type Stream interface {
	// XAdd adds an entry to a stream.
	XAdd(ctx context.Context, key string, id string, values map[string][]byte) (string, error)
	// XRead reads entries from a stream.
	XRead(ctx context.Context, key string, start string, count int64) (collectionx.List[StreamEntry], error)
	// XReadMultiple reads entries from multiple streams.
	XReadMultiple(ctx context.Context, streams map[string]string, count int64, block time.Duration) (collectionx.MultiMap[string, StreamEntry], error)
	// XRange reads entries in a range.
	XRange(ctx context.Context, key string, start, stop string) (collectionx.List[StreamEntry], error)
	// XRevRange reads entries in reverse order.
	XRevRange(ctx context.Context, key string, start, stop string) (collectionx.List[StreamEntry], error)
	// XLen gets the number of entries in a stream.
	XLen(ctx context.Context, key string) (int64, error)
	// XTrim trims the stream to approximately maxLen entries.
	XTrim(ctx context.Context, key string, maxLen int64) error
	// XDel deletes specific entries from a stream.
	XDel(ctx context.Context, key string, ids []string) error

	// Consumer Group Operations
	// XGroupCreate creates a consumer group.
	XGroupCreate(ctx context.Context, key string, group string, startID string) error
	// XGroupDestroy destroys a consumer group.
	XGroupDestroy(ctx context.Context, key string, group string) error
	// XGroupCreateConsumer creates a consumer in a group.
	XGroupCreateConsumer(ctx context.Context, key string, group string, consumer string) error
	// XGroupDelConsumer deletes a consumer from a group.
	XGroupDelConsumer(ctx context.Context, key string, group string, consumer string) error
	// XReadGroup reads entries as part of a consumer group.
	XReadGroup(ctx context.Context, group string, consumer string, streams map[string]string, count int64, block time.Duration) (collectionx.MultiMap[string, StreamEntry], error)
	// XAck acknowledges processing of stream entries.
	XAck(ctx context.Context, key string, group string, ids []string) error
	// XPending gets pending entries information.
	XPending(ctx context.Context, key string, group string) (*PendingInfo, error)
	// XPendingRange gets pending entries in a range.
	XPendingRange(ctx context.Context, key string, group string, start string, stop string, count int64) (collectionx.List[PendingEntry], error)
	// XClaim claims pending entries for a consumer.
	XClaim(ctx context.Context, key string, group string, consumer string, minIdleTime time.Duration, ids []string) (collectionx.List[StreamEntry], error)
	// XAutoClaim auto-claims pending entries.
	XAutoClaim(ctx context.Context, key string, group string, consumer string, minIdleTime time.Duration, start string, count int64) (string, collectionx.List[StreamEntry], error)
	// XInfoGroups gets info about consumer groups.
	XInfoGroups(ctx context.Context, key string) (collectionx.List[GroupInfo], error)
	// XInfoConsumers gets info about consumers in a group.
	XInfoConsumers(ctx context.Context, key string, group string) (collectionx.List[ConsumerInfo], error)
	// XInfoStream gets info about a stream.
	XInfoStream(ctx context.Context, key string) (*StreamInfo, error)
}

// StreamEntry represents a stream entry.
type StreamEntry struct {
	ID     string
	Values collectionx.Map[string, []byte]
}

// PendingInfo represents pending entries summary.
type PendingInfo struct {
	Count     int64
	StartID   string
	EndID     string
	Consumers collectionx.Map[string, int64] // consumer name -> pending count
}

// PendingEntry represents a single pending entry.
type PendingEntry struct {
	ID         string
	Consumer   string
	IdleTime   time.Duration
	Deliveries int64
}

// GroupInfo represents consumer group information.
type GroupInfo struct {
	Name            string
	Consumers       int64
	Pending         int64
	LastDeliveredID string
}

// ConsumerInfo represents consumer information.
type ConsumerInfo struct {
	Name    string
	Pending int64
	Idle    time.Duration
}

// StreamInfo represents stream information.
type StreamInfo struct {
	Length          int64
	RedisVersion    string
	RadixTreeKeys   int64
	RadixTreeNodes  int64
	Groups          int64
	LastGeneratedID string
	FirstEntry      *StreamEntry
	LastEntry       *StreamEntry
}

// Script represents Lua script operations.
type Script interface {
	// Load loads a script into the script cache.
	Load(ctx context.Context, script string) (string, error)
	// Eval executes a script.
	Eval(ctx context.Context, script string, keys []string, args [][]byte) ([]byte, error)
	// EvalSHA executes a cached script by SHA.
	EvalSHA(ctx context.Context, sha string, keys []string, args [][]byte) ([]byte, error)
}

// JSON represents JSON document operations.
type JSON interface {
	// JSONSet sets a JSON value at key.
	JSONSet(ctx context.Context, key string, path string, value []byte, expiration time.Duration) error
	// JSONGet gets a JSON value at key.
	JSONGet(ctx context.Context, key string, path string) ([]byte, error)
	// JSONSetField sets a field in a JSON document.
	JSONSetField(ctx context.Context, key string, path string, value []byte) error
	// JSONGetField gets a field from a JSON document.
	JSONGetField(ctx context.Context, key string, path string) ([]byte, error)
	// JSONDelete deletes a JSON value or field.
	JSONDelete(ctx context.Context, key string, path string) error
}

// Search represents secondary index search operations.
type Search interface {
	// CreateIndex creates a secondary index.
	CreateIndex(ctx context.Context, indexName string, prefix string, schema []SchemaField) error
	// DropIndex drops a secondary index.
	DropIndex(ctx context.Context, indexName string) error
	// Search performs a search query.
	Search(ctx context.Context, indexName string, query string, limit int) (collectionx.List[string], error)
	// SearchWithSort performs a search query with sorting.
	SearchWithSort(ctx context.Context, indexName string, query string, sortBy string, ascending bool, limit int) (collectionx.List[string], error)
	// SearchAggregate performs an aggregation query.
	SearchAggregate(ctx context.Context, indexName string, query string, limit int) ([]map[string]any, error)
}

// Pipeline represents pipeline (batch) operations.
type Pipeline interface {
	// Enqueue adds a command to the pipeline.
	Enqueue(command string, args ...[]byte) error
	// Exec executes all queued commands.
	Exec(ctx context.Context) ([][]byte, error)
	// Close closes the pipeline.
	Close() error
}

// Lock represents distributed lock operations.
type Lock interface {
	// Acquire tries to acquire a lock.
	Acquire(ctx context.Context, key string, token string, ttl time.Duration) (bool, error)
	// Release releases a lock.
	Release(ctx context.Context, key string, token string) (bool, error)
	// Extend extends the lock TTL.
	Extend(ctx context.Context, key string, token string, ttl time.Duration) (bool, error)
}

// Client is the main client interface combining all capabilities.
type Client interface {
	KV
	Hash
	PubSub
	Stream
	Script
	JSON
	Search
	Lock

	// Pipeline creates a new pipeline.
	Pipeline() Pipeline
	// Close closes the client connection.
	Close() error
}

// ClientOptions contains client configuration options.
type ClientOptions struct {
	// Addrs is the list of addresses to connect to.
	Addrs []string
	// Password is the password for authentication.
	Password string
	// DB is the database number (Redis specific).
	DB int
	// UseTLS enables TLS connection.
	UseTLS bool
	// MasterName is the master name for sentinel (optional).
	MasterName string
	// PoolSize is the maximum number of connections in the pool.
	PoolSize int
	// MinIdleConns is the minimum number of idle connections.
	MinIdleConns int
	// ConnMaxLifetime is the maximum lifetime of a connection.
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime is the maximum idle time of a connection.
	ConnMaxIdleTime time.Duration
	// Logger receives adapter lifecycle logs when Debug is true.
	Logger *slog.Logger
	// Debug enables adapter-level debug logs.
	Debug bool
}

// ClientFactory creates a Client from options.
type ClientFactory func(opts ClientOptions) (Client, error)
