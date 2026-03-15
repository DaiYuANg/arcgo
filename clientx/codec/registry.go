package codec

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/DaiYuANg/archgo/collectionx/mapping"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

var (
	ErrNilCodec         = errors.New("codec is nil")
	ErrCodecNameEmpty   = errors.New("codec name is empty")
	ErrCodecExists      = errors.New("codec already exists")
	ErrCodecNotFound    = errors.New("codec not found")
	ErrUnsupportedValue = errors.New("codec unsupported value type")
)

type Registry struct {
	codecs *mapping.ConcurrentMap[string, Codec]
}

func NewRegistry(codecs ...Codec) *Registry {
	r := &Registry{codecs: mapping.NewConcurrentMap[string, Codec]()}
	lo.ForEach(codecs, func(c Codec, _ int) {
		_ = r.Register(c)
	})
	return r
}

func (r *Registry) Register(c Codec) error {
	if c == nil {
		return ErrNilCodec
	}
	name := strings.TrimSpace(strings.ToLower(c.Name()))
	if name == "" {
		return ErrCodecNameEmpty
	}

	if _, loaded := r.codecs.GetOrStore(name, c); loaded {
		return fmt.Errorf("%w: %s", ErrCodecExists, name)
	}
	return nil
}

func (r *Registry) Get(name string) (Codec, bool) {
	c, ok := r.codecs.Get(strings.TrimSpace(strings.ToLower(name)))
	return c, ok
}

func (r *Registry) GetOption(name string) mo.Option[Codec] {
	c, ok := r.Get(name)
	if !ok {
		return mo.None[Codec]()
	}
	return mo.Some(c)
}

func (r *Registry) Must(name string) Codec {
	c, ok := r.GetOption(name).Get()
	if !ok {
		panic(fmt.Sprintf("%v: %s", ErrCodecNotFound, name))
	}
	return c
}

func (r *Registry) Names() []string {
	names := r.codecs.Keys()
	sort.Strings(names)
	return names
}

var defaultRegistry = NewRegistry(
	JSON,
	Text,
	Bytes,
)

func Register(c Codec) error {
	return defaultRegistry.Register(c)
}

func Get(name string) (Codec, bool) {
	return defaultRegistry.Get(name)
}

func GetOption(name string) mo.Option[Codec] {
	return defaultRegistry.GetOption(name)
}

func Must(name string) Codec {
	return defaultRegistry.Must(name)
}

func Names() []string {
	return defaultRegistry.Names()
}
