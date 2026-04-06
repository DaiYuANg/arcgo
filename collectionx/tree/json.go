package tree

import (
	"fmt"

	common "github.com/DaiYuANg/arcgo/collectionx/internal"
)

type jsonNode[K comparable, V any] struct {
	ID       K                `json:"id"`
	Value    V                `json:"value"`
	Children []jsonNode[K, V] `json:"children,omitempty"`
}

// ToJSON serializes tree roots and descendants to JSON.
func (t *Tree[K, V]) ToJSON() ([]byte, error) {
	return marshalTreeJSON("tree", t.toJSONNodes())
}

// MarshalJSON implements json.Marshaler.
func (t *Tree[K, V]) MarshalJSON() ([]byte, error) {
	return forwardTreeJSON("tree", t.ToJSON)
}

// String implements fmt.Stringer.
func (t *Tree[K, V]) String() string {
	return common.StringFromToJSON(t.ToJSON, "[]")
}

// ToJSON serializes concurrent tree snapshot to JSON.
func (t *ConcurrentTree[K, V]) ToJSON() ([]byte, error) {
	return t.Snapshot().ToJSON()
}

// MarshalJSON implements json.Marshaler.
func (t *ConcurrentTree[K, V]) MarshalJSON() ([]byte, error) {
	return forwardTreeJSON("concurrent tree", t.ToJSON)
}

// String implements fmt.Stringer.
func (t *ConcurrentTree[K, V]) String() string {
	return common.StringFromToJSON(t.ToJSON, "[]")
}

func (t *Tree[K, V]) toJSONNodes() []jsonNode[K, V] {
	if t == nil || t.IsEmpty() {
		return nil
	}

	roots := make([]jsonNode[K, V], 0, t.roots.Len())
	rootCount := t.roots.Len()
	for index := range rootCount {
		root, _ := t.roots.Get(index)
		roots = append(roots, toJSONNode(root))
	}
	return roots
}

func toJSONNode[K comparable, V any](node *Node[K, V]) jsonNode[K, V] {
	if node == nil {
		return jsonNode[K, V]{}
	}

	jsonNodeValue := jsonNode[K, V]{
		ID:    node.ID(),
		Value: node.Value(),
	}
	if node.children.Len() == 0 {
		return jsonNodeValue
	}

	jsonNodeValue.Children = make([]jsonNode[K, V], 0, node.children.Len())
	childCount := node.children.Len()
	for index := range childCount {
		child, _ := node.children.Get(index)
		jsonNodeValue.Children = append(jsonNodeValue.Children, toJSONNode(child))
	}
	return jsonNodeValue
}

func marshalTreeJSON[T any](kind string, value T) ([]byte, error) {
	data, err := common.MarshalJSONValue(value)
	if err != nil {
		return nil, fmt.Errorf("marshal %s JSON: %w", kind, err)
	}

	return data, nil
}

func forwardTreeJSON(kind string, fn func() ([]byte, error)) ([]byte, error) {
	data, err := common.ForwardToJSON(fn)
	if err != nil {
		return nil, fmt.Errorf("marshal %s JSON: %w", kind, err)
	}

	return data, nil
}
