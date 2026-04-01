package repository

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

type hashRepositoryConfig[T any] struct {
	keyBuilder *mapping.KeyBuilder
	tagParser  *mapping.TagParser
	codec      *mapping.HashCodec
	indexer    *Indexer[T]
	pipeline   mo.Option[pipelineProvider]
	script     mo.Option[kvx.Script]
	logger     *slog.Logger
	debug      bool
}

type jsonRepositoryConfig[T any] struct {
	keyBuilder *mapping.KeyBuilder
	tagParser  *mapping.TagParser
	serializer mapping.Serializer
	indexer    *Indexer[T]
	pipeline   mo.Option[pipelineProvider]
	script     mo.Option[kvx.Script]
	logger     *slog.Logger
	debug      bool
}

// HashRepositoryOption applies configuration to a hash repository.
type HashRepositoryOption[T any] interface {
	applyHash(*hashRepositoryConfig[T])
}

// JSONRepositoryOption applies configuration to a JSON repository.
type JSONRepositoryOption[T any] interface {
	applyJSON(*jsonRepositoryConfig[T])
}

// Option applies configuration to both hash and JSON repositories.
type Option[T any] interface {
	HashRepositoryOption[T]
	JSONRepositoryOption[T]
}

var (
	_ HashRepositoryOption[struct{}] = hashOptionFunc[struct{}](nil)
	_ JSONRepositoryOption[struct{}] = jsonOptionFunc[struct{}](nil)
	_ Option[struct{}]               = dualOption[struct{}]{}
	_                                = hashOptionFunc[struct{}](nil).applyHash
	_                                = jsonOptionFunc[struct{}](nil).applyJSON
	_                                = dualOption[struct{}]{}.applyHash
	_                                = dualOption[struct{}]{}.applyJSON
)

type hashOptionFunc[T any] func(*hashRepositoryConfig[T])

func (f hashOptionFunc[T]) applyHash(cfg *hashRepositoryConfig[T]) { f(cfg) }

type jsonOptionFunc[T any] func(*jsonRepositoryConfig[T])

func (f jsonOptionFunc[T]) applyJSON(cfg *jsonRepositoryConfig[T]) { f(cfg) }

type dualOption[T any] struct {
	hash hashOptionFunc[T]
	json jsonOptionFunc[T]
}

func (o dualOption[T]) applyHash(cfg *hashRepositoryConfig[T]) {
	if o.hash != nil {
		o.hash(cfg)
	}
}

func (o dualOption[T]) applyJSON(cfg *jsonRepositoryConfig[T]) {
	if o.json != nil {
		o.json(cfg)
	}
}

// OptionSet lets callers reuse a standard repository configuration across entities.
type OptionSet[T any] struct {
	Hash []HashRepositoryOption[T]
	JSON []JSONRepositoryOption[T]
}

// HashOptions returns hash repository options plus any extra options.
func (s OptionSet[T]) HashOptions(extra ...HashRepositoryOption[T]) []HashRepositoryOption[T] {
	return lo.Concat(s.Hash, extra)
}

// JSONOptions returns JSON repository options plus any extra options.
func (s OptionSet[T]) JSONOptions(extra ...JSONRepositoryOption[T]) []JSONRepositoryOption[T] {
	return lo.Concat(s.JSON, extra)
}

// Preset groups reusable repository options.
type Preset[T any] struct {
	Hash []HashRepositoryOption[T]
	JSON []JSONRepositoryOption[T]
}

// HashOptions returns preset hash repository options plus any extra options.
func (p Preset[T]) HashOptions(extra ...HashRepositoryOption[T]) []HashRepositoryOption[T] {
	return lo.Concat(p.Hash, extra)
}

// JSONOptions returns preset JSON repository options plus any extra options.
func (p Preset[T]) JSONOptions(extra ...JSONRepositoryOption[T]) []JSONRepositoryOption[T] {
	return lo.Concat(p.JSON, extra)
}

func defaultHashConfig[T any](kv kvx.KV, keyPrefix string) hashRepositoryConfig[T] {
	return hashRepositoryConfig[T]{
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
		tagParser:  mapping.NewTagParser(),
		codec:      mapping.NewHashCodec(nil),
		indexer:    NewIndexer[T](kv, keyPrefix),
		pipeline:   mo.None[pipelineProvider](),
		script:     mo.None[kvx.Script](),
		logger:     slog.Default(),
	}
}

func defaultJSONConfig[T any](kv kvx.KV, keyPrefix string) jsonRepositoryConfig[T] {
	return jsonRepositoryConfig[T]{
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
		tagParser:  mapping.NewTagParser(),
		serializer: mapping.NewJSONSerializer(),
		indexer:    NewIndexer[T](kv, keyPrefix),
		pipeline:   mo.None[pipelineProvider](),
		script:     mo.None[kvx.Script](),
		logger:     slog.Default(),
	}
}

func applyHashOptions[T any](cfg *hashRepositoryConfig[T], options ...HashRepositoryOption[T]) {
	lo.ForEach(lo.Filter(options, func(option HashRepositoryOption[T], _ int) bool {
		return option != nil
	}), func(option HashRepositoryOption[T], _ int) {
		option.applyHash(cfg)
	})
}

func applyJSONOptions[T any](cfg *jsonRepositoryConfig[T], options ...JSONRepositoryOption[T]) {
	lo.ForEach(lo.Filter(options, func(option JSONRepositoryOption[T], _ int) bool {
		return option != nil
	}), func(option JSONRepositoryOption[T], _ int) {
		option.applyJSON(cfg)
	})
}

// WithPipeline configures pipeline support for both repository backends.
func WithPipeline[T any](provider pipelineProvider) Option[T] {
	return dualOption[T]{
		hash: func(cfg *hashRepositoryConfig[T]) { cfg.pipeline = mo.Some[pipelineProvider](provider) },
		json: func(cfg *jsonRepositoryConfig[T]) { cfg.pipeline = mo.Some[pipelineProvider](provider) },
	}
}

// WithScript configures Lua script support for both repository backends.
func WithScript[T any](script kvx.Script) Option[T] {
	return dualOption[T]{
		hash: func(cfg *hashRepositoryConfig[T]) {
			if script != nil {
				cfg.script = mo.Some(script)
			}
		},
		json: func(cfg *jsonRepositoryConfig[T]) {
			if script != nil {
				cfg.script = mo.Some(script)
			}
		},
	}
}

// WithKeyBuilder configures a custom key builder for both repository backends.
func WithKeyBuilder[T any](builder *mapping.KeyBuilder) Option[T] {
	return dualOption[T]{
		hash: func(cfg *hashRepositoryConfig[T]) {
			if builder != nil {
				cfg.keyBuilder = builder
			}
		},
		json: func(cfg *jsonRepositoryConfig[T]) {
			if builder != nil {
				cfg.keyBuilder = builder
			}
		},
	}
}

// WithTagParser configures a custom tag parser for both repository backends.
func WithTagParser[T any](parser *mapping.TagParser) Option[T] {
	return dualOption[T]{
		hash: func(cfg *hashRepositoryConfig[T]) {
			if parser != nil {
				cfg.tagParser = parser
			}
		},
		json: func(cfg *jsonRepositoryConfig[T]) {
			if parser != nil {
				cfg.tagParser = parser
			}
		},
	}
}

// WithIndexer configures a custom secondary indexer for both repository backends.
func WithIndexer[T any](indexer *Indexer[T]) Option[T] {
	return dualOption[T]{
		hash: func(cfg *hashRepositoryConfig[T]) {
			if indexer != nil {
				cfg.indexer = indexer
			}
		},
		json: func(cfg *jsonRepositoryConfig[T]) {
			if indexer != nil {
				cfg.indexer = indexer
			}
		},
	}
}

// WithHashCodec configures a custom hash codec for hash repositories.
func WithHashCodec[T any](codec *mapping.HashCodec) HashRepositoryOption[T] {
	return hashOptionFunc[T](func(cfg *hashRepositoryConfig[T]) {
		if codec != nil {
			cfg.codec = codec
		}
	})
}

// WithSerializer configures a custom serializer for JSON repositories.
func WithSerializer[T any](serializer mapping.Serializer) JSONRepositoryOption[T] {
	return jsonOptionFunc[T](func(cfg *jsonRepositoryConfig[T]) {
		if serializer != nil {
			cfg.serializer = serializer
		}
	})
}

// NewPreset creates a reusable repository preset from shared repository options.
func NewPreset[T any](options ...Option[T]) Preset[T] {
	return Preset[T]{
		Hash: lo.Map(options, func(option Option[T], _ int) HashRepositoryOption[T] { return option }),
		JSON: lo.Map(options, func(option Option[T], _ int) JSONRepositoryOption[T] { return option }),
	}
}

// WithLogger configures structured logging for both repository backends.
func WithLogger[T any](logger *slog.Logger) Option[T] {
	return dualOption[T]{
		hash: func(cfg *hashRepositoryConfig[T]) {
			if logger != nil {
				cfg.logger = logger
			}
		},
		json: func(cfg *jsonRepositoryConfig[T]) {
			if logger != nil {
				cfg.logger = logger
			}
		},
	}
}

// WithDebug enables or disables debug logging for both repository backends.
func WithDebug[T any](enabled bool) Option[T] {
	return dualOption[T]{
		hash: func(cfg *hashRepositoryConfig[T]) { cfg.debug = enabled },
		json: func(cfg *jsonRepositoryConfig[T]) { cfg.debug = enabled },
	}
}
