package dbx

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/pkg/option"
	"github.com/samber/lo"
)

type MapperOption func(*mapperBuildOptions) error

type mapperBuildOptions struct {
	runtime *mapperRuntime
}

func WithMapperCodecs(codecs ...Codec) MapperOption {
	return func(opts *mapperBuildOptions) error {
		filtered := lo.Filter(codecs, func(codec Codec, _ int) bool {
			return !isNilCodec(codec)
		})
		if len(filtered) == 0 {
			return nil
		}

		runtime := opts.runtime.clone()
		for _, codec := range filtered {
			if err := runtime.codecs.register(codec); err != nil {
				return err
			}
		}
		opts.runtime = runtime
		return nil
	}
}

func defaultMapperBuildOptions() mapperBuildOptions {
	return mapperBuildOptions{
		runtime: defaultMapperRuntime,
	}
}

func applyMapperOptions(opts ...MapperOption) (mapperBuildOptions, error) {
	config := defaultMapperBuildOptions()
	if err := option.ApplyErr(&config, opts...); err != nil {
		return mapperBuildOptions{}, fmt.Errorf("dbx: apply mapper options: %w", err)
	}
	return config, nil
}

func (r *mapperRuntime) clone() *mapperRuntime {
	if r == nil {
		return newMapperRuntime()
	}
	cloned := &mapperRuntime{
		registry: newMapperRegistry(),
		codecs:   r.codecs.clone(),
	}
	return cloned
}
