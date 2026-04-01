package repository

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

// HashRepository provides repository operations for hash-backed entities.
type HashRepository[T any] struct {
	base     repositoryBase[T]
	client   kvx.Hash
	kv       kvx.KV
	pipeline mo.Option[pipelineProvider]
	script   mo.Option[kvx.Script]
	codec    *mapping.HashCodec
	logger   *slog.Logger
	debug    bool
}

// NewHashRepository creates a hash-backed repository for entity type T.
func NewHashRepository[T any](client kvx.Hash, kv kvx.KV, keyPrefix string, options ...HashRepositoryOption[T]) *HashRepository[T] {
	cfg := defaultHashConfig[T](kv, keyPrefix)
	applyHashOptions(&cfg, options...)

	repo := &HashRepository[T]{
		base: repositoryBase[T]{
			keyBuilder: cfg.keyBuilder,
			tagParser:  cfg.tagParser,
			indexer:    cfg.indexer,
		},
		client:   client,
		kv:       kv,
		pipeline: cfg.pipeline,
		script:   cfg.script,
		codec:    cfg.codec,
		logger:   cfg.logger,
		debug:    cfg.debug,
	}
	repo.logDebug("kvx hash repository created", "key_prefix", keyPrefix)
	return repo
}

// NewHashRepositoryWithClient creates a hash-backed repository using a full kvx client.
func NewHashRepositoryWithClient[T any](client kvx.Client, keyPrefix string, options ...HashRepositoryOption[T]) *HashRepository[T] {
	return NewHashRepository[T](client, client, keyPrefix, lo.Concat(
		[]HashRepositoryOption[T]{WithPipeline[T](client), WithScript[T](client)},
		options,
	)...)
}

// NewHashRepositoryWithCodec creates a hash-backed repository with a custom hash codec.
func NewHashRepositoryWithCodec[T any](client kvx.Hash, kv kvx.KV, keyPrefix string, codec *mapping.HashCodec, options ...HashRepositoryOption[T]) *HashRepository[T] {
	return NewHashRepository[T](client, kv, keyPrefix, lo.Concat(
		[]HashRepositoryOption[T]{WithHashCodec[T](codec)},
		options,
	)...)
}

func (r *HashRepository[T]) logDebug(msg string, attrs ...any) {
	kvx.LogDebug(r.logger, r.debug, msg, attrs...)
}

func (r *HashRepository[T]) logError(msg string, attrs ...any) {
	kvx.LogError(r.logger, msg, attrs...)
}
