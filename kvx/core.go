// Package kvx provides a layered KV abstraction with Redis/Valkey support.
//
// Architecture:
//   - Layer 1: Core Client Abstraction (KV, Hash, PubSub, Stream, etc.)
//   - Layer 2: Mapping Layer (tag parser, metadata, serializers)
//   - Layer 3: Repository Layer (HashRepository, JSONRepository, etc.)
//   - Layer 4: Feature Modules (pubsub, stream, json, search, script)
//   - Layer 5: Adapters (redis, valkey)
package kvx

import (
	"context"
	"time"
)

// KV is the base key-value interface.
type KV interface {
	// Get retrieves the value for the given key.
	Get(ctx context.Context, key string) ([]byte, error)
	// Set sets the value for the given key.
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	// Delete deletes the given key.
	Delete(ctx context.Context, key string) error
	// Exists checks if the key exists.
	Exists(ctx context.Context, key string) (bool, error)
	// Expire sets the expiration for the given key.
	Expire(ctx context.Context, key string, expiration time.Duration) error
	// TTL gets the TTL for the given key.
	TTL(ctx context.Context, key string) (time.Duration, error)
}

// Hash represents a hash (field-value map) operation.
type Hash interface {
	// HGet gets a field from a hash.
	HGet(ctx context.Context, key string, field string) ([]byte, error)
	// HSet sets fields in a hash.
	HSet(ctx context.Context, key string, values map[string][]byte) error
	// HGetAll gets all fields and values from a hash.
	HGetAll(ctx context.Context, key string) (map[string][]byte, error)
	// HDel deletes fields from a hash.
	HDel(ctx context.Context, key string, fields ...string) error
	// HExists checks if a field exists in a hash.
	HExists(ctx context.Context, key string, field string) (bool, error)
	// HKeys gets all field names in a hash.
	HKeys(ctx context.Context, key string) ([]string, error)
	// HVals gets all values in a hash.
	HVals(ctx context.Context, key string) ([][]byte, error)
	// HLen gets the number of fields in a hash.
	HLen(ctx context.Context, key string) (int64, error)
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
	XRead(ctx context.Context, key string, start string, count int64) ([]StreamEntry, error)
	// XRange reads entries in a range.
	XRange(ctx context.Context, key string, start, stop string) ([]StreamEntry, error)
	// XLen gets the number of entries in a stream.
	XLen(ctx context.Context, key string) (int64, error)
	// XTrim trims the stream to approximately maxLen entries.
	XTrim(ctx context.Context, key string, maxLen int64) error
}

// StreamEntry represents a stream entry.
type StreamEntry struct {
	ID     string
	Values map[string][]byte
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
	Search(ctx context.Context, indexName string, query string, limit int) ([]string, error)
}

// SchemaField represents a search schema field.
type SchemaField struct {
	Name     string
	Type     SchemaFieldType
	Indexing bool
	Sortable bool
}

// SchemaFieldType represents the type of a schema field.
type SchemaField string

const (
	SchemaFieldTypeText    SchemaFieldType = "TEXT"
	SchemaFieldTypeTag     SchemaFieldType = "TAG"
	SchemaFieldTypeNumeric SchemaFieldType = "NUMERIC"
)

// Pipeline represents pipeline (batch) operations.
type Pipeline interface {
	// Enqueue adds a command to the pipeline.
	Enqueue(command string, args ...[]byte)
	// Exec executes all queued commands.
	Exec(ctx context.Context) ([][]byte, error)
	// Close closes the pipeline.
	Close() error
}

// Lock represents distributed lock operations.
type Lock interface {
	// Acquire tries to acquire a lock.
	Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)
	// Release releases a lock.
	Release(ctx context.Context, key string) error
	// Extend extends the lock TTL.
	Extend(ctx context.Context, key string, ttl time.Duration) (bool, error)
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
	Pipeline
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
}

// ClientFactory creates a Client from options.
type ClientFactory func(opts ClientOptions) (Client, error)
