package repository

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

// JSONRepository provides repository operations for JSON-backed entities.
type JSONRepository[T any] struct {
	base       repositoryBase[T]
	client     kvx.JSON
	kv         kvx.KV
	pipeline   mo.Option[pipelineProvider]
	script     mo.Option[kvx.Script]
	serializer mapping.Serializer
	logger     *slog.Logger
	debug      bool
}

// NewJSONRepository creates a JSON-backed repository for entity type T.
func NewJSONRepository[T any](client kvx.JSON, kv kvx.KV, keyPrefix string, options ...JSONRepositoryOption[T]) *JSONRepository[T] {
	cfg := defaultJSONConfig[T](kv, keyPrefix)
	applyJSONOptions(&cfg, options...)

	repo := &JSONRepository[T]{
		base: repositoryBase[T]{
			keyBuilder: cfg.keyBuilder,
			tagParser:  cfg.tagParser,
			indexer:    cfg.indexer,
		},
		client:     client,
		kv:         kv,
		pipeline:   cfg.pipeline,
		script:     cfg.script,
		serializer: cfg.serializer,
		logger:     cfg.logger,
		debug:      cfg.debug,
	}
	repo.logDebug("kvx json repository created", "key_prefix", keyPrefix)
	return repo
}

// NewJSONRepositoryWithClient creates a JSON-backed repository using a full kvx client.
func NewJSONRepositoryWithClient[T any](client kvx.Client, keyPrefix string, options ...JSONRepositoryOption[T]) *JSONRepository[T] {
	return NewJSONRepository[T](client, client, keyPrefix, lo.Concat(
		[]JSONRepositoryOption[T]{WithPipeline[T](client), WithScript[T](client)},
		options,
	)...)
}

func (r *JSONRepository[T]) logDebug(msg string, attrs ...any) {
	kvx.LogDebug(r.logger, r.debug, msg, attrs...)
}

func (r *JSONRepository[T]) logError(msg string, attrs ...any) {
	kvx.LogError(r.logger, msg, attrs...)
}
