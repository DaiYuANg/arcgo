package json

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
)

// DocumentRepository provides repository-style access to JSON documents.
type DocumentRepository[T any] struct {
	json      *JSON
	keyPrefix string
}

// NewDocumentRepository creates a new DocumentRepository.
func NewDocumentRepository[T any](client kvx.JSON, keyPrefix string) *DocumentRepository[T] {
	return &DocumentRepository[T]{
		json:      NewJSON(client),
		keyPrefix: keyPrefix,
	}
}

// buildKey builds the full key for a document.
func (r *DocumentRepository[T]) buildKey(id string) string {
	if r.keyPrefix == "" {
		return id
	}
	return r.keyPrefix + ":" + id
}

// Save saves a document.
func (r *DocumentRepository[T]) Save(ctx context.Context, id string, doc *T, expiration time.Duration) error {
	key := r.buildKey(id)
	return r.json.Set(ctx, key, doc, expiration)
}

// FindByID finds a document by ID.
func (r *DocumentRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	key := r.buildKey(id)
	var doc T
	if err := r.json.Get(ctx, key, &doc); err != nil {
		return nil, fmt.Errorf("find document %q: %w", key, err)
	}
	return &doc, nil
}

// Delete deletes a document.
func (r *DocumentRepository[T]) Delete(ctx context.Context, id string) error {
	key := r.buildKey(id)
	return r.json.Delete(ctx, key)
}

// Exists checks if a document exists.
func (r *DocumentRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	key := r.buildKey(id)
	return r.json.Exists(ctx, key)
}

// UpdatePath updates a specific path in a document.
func (r *DocumentRepository[T]) UpdatePath(ctx context.Context, id, path string, value any) error {
	key := r.buildKey(id)
	return r.json.SetPath(ctx, key, path, value)
}

// GetPath gets a specific path from a document.
func (r *DocumentRepository[T]) GetPath(ctx context.Context, id, path string, dest any) error {
	key := r.buildKey(id)
	return r.json.GetPath(ctx, key, path, dest)
}
