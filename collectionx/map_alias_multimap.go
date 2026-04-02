package collectionx

import (
	"github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/DaiYuANg/arcgo/collectionx/mapping"
)

type multiMapFluent[K comparable, V any] interface {
	WhereKeys(predicate func(key K, values []V) bool) *mapping.MultiMap[K, V]
	RejectKeys(predicate func(key K, values []V) bool) *mapping.MultiMap[K, V]
	WhereValues(predicate func(key K, value V) bool) *mapping.MultiMap[K, V]
	RejectValues(predicate func(key K, value V) bool) *mapping.MultiMap[K, V]
	EachKey(fn func(key K, values []V)) *mapping.MultiMap[K, V]
	EachValue(fn func(key K, value V)) *mapping.MultiMap[K, V]
	FirstValueWhere(predicate func(key K, value V) bool) (K, V, bool)
	AnyValueMatch(predicate func(key K, value V) bool) bool
	AllValuesMatch(predicate func(key K, value V) bool) bool
	FlattenValues() *list.List[V]
}
