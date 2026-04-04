package list

import (
	"fmt"

	common "github.com/DaiYuANg/arcgo/collectionx/internal"
)

// ToJSON serializes list values to JSON.
func (l *List[T]) ToJSON() ([]byte, error) {
	return marshalListJSON(l.Values(), "list")
}

// MarshalJSON implements json.Marshaler.
func (l *List[T]) MarshalJSON() ([]byte, error) {
	return forwardListJSON(l.ToJSON, "list")
}

// String implements fmt.Stringer.
func (l *List[T]) String() string {
	return common.StringFromToJSON(l.ToJSON, "[]")
}

// ToJSON serializes grid rows to JSON.
func (g *Grid[T]) ToJSON() ([]byte, error) {
	return marshalListJSON(g.Values(), "grid")
}

// MarshalJSON implements json.Marshaler.
func (g *Grid[T]) MarshalJSON() ([]byte, error) {
	return forwardListJSON(g.ToJSON, "grid")
}

// String implements fmt.Stringer.
func (g *Grid[T]) String() string {
	return common.StringFromToJSON(g.ToJSON, "[]")
}

// ToJSON serializes concurrent grid rows to JSON.
func (g *ConcurrentGrid[T]) ToJSON() ([]byte, error) {
	return marshalListJSON(g.Values(), "concurrent grid")
}

// MarshalJSON implements json.Marshaler.
func (g *ConcurrentGrid[T]) MarshalJSON() ([]byte, error) {
	return forwardListJSON(g.ToJSON, "concurrent grid")
}

// String implements fmt.Stringer.
func (g *ConcurrentGrid[T]) String() string {
	return common.StringFromToJSON(g.ToJSON, "[]")
}

// ToJSON serializes concurrent list values to JSON.
func (l *ConcurrentList[T]) ToJSON() ([]byte, error) {
	return marshalListJSON(l.Values(), "concurrent list")
}

// MarshalJSON implements json.Marshaler.
func (l *ConcurrentList[T]) MarshalJSON() ([]byte, error) {
	return forwardListJSON(l.ToJSON, "concurrent list")
}

// String implements fmt.Stringer.
func (l *ConcurrentList[T]) String() string {
	return common.StringFromToJSON(l.ToJSON, "[]")
}

// ToJSON serializes deque values to JSON.
func (d *Deque[T]) ToJSON() ([]byte, error) {
	return marshalListJSON(d.Values(), "deque")
}

// MarshalJSON implements json.Marshaler.
func (d *Deque[T]) MarshalJSON() ([]byte, error) {
	return forwardListJSON(d.ToJSON, "deque")
}

// String implements fmt.Stringer.
func (d *Deque[T]) String() string {
	return common.StringFromToJSON(d.ToJSON, "[]")
}

// ToJSON serializes concurrent-deque values to JSON.
func (d *ConcurrentDeque[T]) ToJSON() ([]byte, error) {
	return marshalListJSON(d.Values(), "concurrent deque")
}

// MarshalJSON implements json.Marshaler.
func (d *ConcurrentDeque[T]) MarshalJSON() ([]byte, error) {
	return forwardListJSON(d.ToJSON, "concurrent deque")
}

// String implements fmt.Stringer.
func (d *ConcurrentDeque[T]) String() string {
	return common.StringFromToJSON(d.ToJSON, "[]")
}

// ToJSON serializes ring-buffer values to JSON.
func (r *RingBuffer[T]) ToJSON() ([]byte, error) {
	return marshalListJSON(r.Values(), "ring buffer")
}

// MarshalJSON implements json.Marshaler.
func (r *RingBuffer[T]) MarshalJSON() ([]byte, error) {
	return forwardListJSON(r.ToJSON, "ring buffer")
}

// String implements fmt.Stringer.
func (r *RingBuffer[T]) String() string {
	return common.StringFromToJSON(r.ToJSON, "[]")
}

// ToJSON serializes concurrent-ring-buffer values to JSON.
func (r *ConcurrentRingBuffer[T]) ToJSON() ([]byte, error) {
	return marshalListJSON(r.Values(), "concurrent ring buffer")
}

// MarshalJSON implements json.Marshaler.
func (r *ConcurrentRingBuffer[T]) MarshalJSON() ([]byte, error) {
	return forwardListJSON(r.ToJSON, "concurrent ring buffer")
}

// String implements fmt.Stringer.
func (r *ConcurrentRingBuffer[T]) String() string {
	return common.StringFromToJSON(r.ToJSON, "[]")
}

// ToJSON serializes rope list values to JSON.
func (r *RopeList[T]) ToJSON() ([]byte, error) {
	return marshalListJSON(r.Values(), "rope list")
}

// MarshalJSON implements json.Marshaler.
func (r *RopeList[T]) MarshalJSON() ([]byte, error) {
	return forwardListJSON(r.ToJSON, "rope list")
}

// String implements fmt.Stringer.
func (r *RopeList[T]) String() string {
	return common.StringFromToJSON(r.ToJSON, "[]")
}

// ToJSON serializes priority queue values to JSON in sorted priority order.
func (pq *PriorityQueue[T]) ToJSON() ([]byte, error) {
	return marshalListJSON(pq.ValuesSorted(), "priority queue")
}

// MarshalJSON implements json.Marshaler.
func (pq *PriorityQueue[T]) MarshalJSON() ([]byte, error) {
	return forwardListJSON(pq.ToJSON, "priority queue")
}

// String implements fmt.Stringer.
func (pq *PriorityQueue[T]) String() string {
	return common.StringFromToJSON(pq.ToJSON, "[]")
}

func marshalListJSON(value any, kind string) ([]byte, error) {
	data, err := common.MarshalJSONValue(value)
	if err != nil {
		return nil, fmt.Errorf("marshal %s json: %w", kind, err)
	}
	return data, nil
}

func forwardListJSON(toJSON func() ([]byte, error), kind string) ([]byte, error) {
	data, err := common.ForwardToJSON(toJSON)
	if err != nil {
		return nil, fmt.Errorf("marshal %s: %w", kind, err)
	}
	return data, nil
}
