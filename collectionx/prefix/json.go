package prefix

import (
	"fmt"

	common "github.com/DaiYuANg/arcgo/collectionx/internal"
	"github.com/samber/lo"
)

// All returns all key-value pairs as a copied map.
func (t *Trie[V]) All() map[string]V {
	pairs := t.pairsWithPrefix("")
	if len(pairs) == 0 {
		return map[string]V{}
	}

	return lo.Associate(pairs, func(item keyValue[V]) (string, V) {
		return item.key, item.value
	})
}

// ToJSON serializes all key-value pairs to JSON.
func (t *Trie[V]) ToJSON() ([]byte, error) {
	data, err := common.MarshalJSONValue(t.All())
	if err != nil {
		return nil, fmt.Errorf("marshal trie json: %w", err)
	}
	return data, nil
}

// MarshalJSON implements json.Marshaler.
func (t *Trie[V]) MarshalJSON() ([]byte, error) {
	data, err := common.ForwardToJSON(t.ToJSON)
	if err != nil {
		return nil, fmt.Errorf("marshal trie: %w", err)
	}
	return data, nil
}

// String implements fmt.Stringer.
func (t *Trie[V]) String() string {
	return common.StringFromToJSON(t.ToJSON, "{}")
}
