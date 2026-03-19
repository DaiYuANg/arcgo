package dbx

import (
	"reflect"

	"github.com/DaiYuANg/arcgo/collectionx"
)

type mapperRegistry struct {
	structMappers collectionx.ConcurrentMap[reflect.Type, *mapperMetadata]
}

type mapperRuntime struct {
	registry *mapperRegistry
	codecs   *codecRegistry
}

var defaultMapperRuntime = newMapperRuntime()

func newMapperRuntime() *mapperRuntime {
	runtime := &mapperRuntime{
		registry: newMapperRegistry(),
		codecs:   newCodecRegistry(),
	}
	registerBuiltinCodecs(runtime.codecs)
	return runtime
}

func newMapperRegistry() *mapperRegistry {
	return &mapperRegistry{
		structMappers: collectionx.NewConcurrentMap[reflect.Type, *mapperMetadata](),
	}
}

func getOrBuildStructMapperMetadata[E any]() (*mapperMetadata, error) {
	return getOrBuildMapperMetadata[E](defaultMapperRuntime)
}

func getOrBuildMapperMetadata[E any](runtime *mapperRuntime) (*mapperMetadata, error) {
	entityType := reflect.TypeFor[E]()
	if cached, ok := runtime.registry.structMappers.Get(entityType); ok {
		return cached, nil
	}

	mapper, err := buildMapperMetadata(entityType, runtime.codecs)
	if err != nil {
		return nil, err
	}
	actual, _ := runtime.registry.structMappers.GetOrStore(entityType, mapper)
	return actual, nil
}
