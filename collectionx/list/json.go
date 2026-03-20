package list

import common "github.com/DaiYuANg/arcgo/collectionx/internal"

// ToJSON serializes list values to JSON.
func (l *List[T]) ToJSON() ([]byte, error) {
	return common.MarshalJSONValue(l.Values())
}

// MarshalJSON implements json.Marshaler.
func (l *List[T]) MarshalJSON() ([]byte, error) {
	return common.ForwardToJSON(l.ToJSON)
}

// String implements fmt.Stringer.
func (l *List[T]) String() string {
	return common.StringFromToJSON(l.ToJSON, "[]")
}

// ToJSON serializes concurrent list values to JSON.
func (l *ConcurrentList[T]) ToJSON() ([]byte, error) {
	return common.MarshalJSONValue(l.Values())
}

// MarshalJSON implements json.Marshaler.
func (l *ConcurrentList[T]) MarshalJSON() ([]byte, error) {
	return common.ForwardToJSON(l.ToJSON)
}

// String implements fmt.Stringer.
func (l *ConcurrentList[T]) String() string {
	return common.StringFromToJSON(l.ToJSON, "[]")
}

// ToJSON serializes deque values to JSON.
func (d *Deque[T]) ToJSON() ([]byte, error) {
	return common.MarshalJSONValue(d.Values())
}

// MarshalJSON implements json.Marshaler.
func (d *Deque[T]) MarshalJSON() ([]byte, error) {
	return common.ForwardToJSON(d.ToJSON)
}

// String implements fmt.Stringer.
func (d *Deque[T]) String() string {
	return common.StringFromToJSON(d.ToJSON, "[]")
}

// ToJSON serializes concurrent-deque values to JSON.
func (d *ConcurrentDeque[T]) ToJSON() ([]byte, error) {
	return common.MarshalJSONValue(d.Values())
}

// MarshalJSON implements json.Marshaler.
func (d *ConcurrentDeque[T]) MarshalJSON() ([]byte, error) {
	return common.ForwardToJSON(d.ToJSON)
}

// String implements fmt.Stringer.
func (d *ConcurrentDeque[T]) String() string {
	return common.StringFromToJSON(d.ToJSON, "[]")
}

// ToJSON serializes ring-buffer values to JSON.
func (r *RingBuffer[T]) ToJSON() ([]byte, error) {
	return common.MarshalJSONValue(r.Values())
}

// MarshalJSON implements json.Marshaler.
func (r *RingBuffer[T]) MarshalJSON() ([]byte, error) {
	return common.ForwardToJSON(r.ToJSON)
}

// String implements fmt.Stringer.
func (r *RingBuffer[T]) String() string {
	return common.StringFromToJSON(r.ToJSON, "[]")
}

// ToJSON serializes concurrent-ring-buffer values to JSON.
func (r *ConcurrentRingBuffer[T]) ToJSON() ([]byte, error) {
	return common.MarshalJSONValue(r.Values())
}

// MarshalJSON implements json.Marshaler.
func (r *ConcurrentRingBuffer[T]) MarshalJSON() ([]byte, error) {
	return common.ForwardToJSON(r.ToJSON)
}

// String implements fmt.Stringer.
func (r *ConcurrentRingBuffer[T]) String() string {
	return common.StringFromToJSON(r.ToJSON, "[]")
}

// ToJSON serializes rope list values to JSON.
func (r *RopeList[T]) ToJSON() ([]byte, error) {
	return common.MarshalJSONValue(r.Values())
}

// MarshalJSON implements json.Marshaler.
func (r *RopeList[T]) MarshalJSON() ([]byte, error) {
	return common.ForwardToJSON(r.ToJSON)
}

// String implements fmt.Stringer.
func (r *RopeList[T]) String() string {
	return common.StringFromToJSON(r.ToJSON, "[]")
}

// ToJSON serializes priority queue values to JSON in sorted priority order.
func (pq *PriorityQueue[T]) ToJSON() ([]byte, error) {
	return common.MarshalJSONValue(pq.ValuesSorted())
}

// MarshalJSON implements json.Marshaler.
func (pq *PriorityQueue[T]) MarshalJSON() ([]byte, error) {
	return common.ForwardToJSON(pq.ToJSON)
}

// String implements fmt.Stringer.
func (pq *PriorityQueue[T]) String() string {
	return common.StringFromToJSON(pq.ToJSON, "[]")
}
