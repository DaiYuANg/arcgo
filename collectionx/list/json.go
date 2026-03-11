package list

import (
	"encoding/json"

	common "github.com/DaiYuANg/arcgo/collectionx/internal"
)

// ToJSON serializes list values to JSON.
func (l *List[T]) ToJSON() ([]byte, error) {
	return json.Marshal(l.Values())
}

// MarshalJSON implements json.Marshaler.
func (l *List[T]) MarshalJSON() ([]byte, error) {
	return l.ToJSON()
}

// String implements fmt.Stringer.
func (l *List[T]) String() string {
	data, err := l.ToJSON()
	return common.JSONResultString(data, err, "[]")
}

// ToJSON serializes concurrent list values to JSON.
func (l *ConcurrentList[T]) ToJSON() ([]byte, error) {
	return json.Marshal(l.Values())
}

// MarshalJSON implements json.Marshaler.
func (l *ConcurrentList[T]) MarshalJSON() ([]byte, error) {
	return l.ToJSON()
}

// String implements fmt.Stringer.
func (l *ConcurrentList[T]) String() string {
	data, err := l.ToJSON()
	return common.JSONResultString(data, err, "[]")
}

// ToJSON serializes deque values to JSON.
func (d *Deque[T]) ToJSON() ([]byte, error) {
	return json.Marshal(d.Values())
}

// MarshalJSON implements json.Marshaler.
func (d *Deque[T]) MarshalJSON() ([]byte, error) {
	return d.ToJSON()
}

// String implements fmt.Stringer.
func (d *Deque[T]) String() string {
	data, err := d.ToJSON()
	return common.JSONResultString(data, err, "[]")
}

// ToJSON serializes concurrent-deque values to JSON.
func (d *ConcurrentDeque[T]) ToJSON() ([]byte, error) {
	return json.Marshal(d.Values())
}

// MarshalJSON implements json.Marshaler.
func (d *ConcurrentDeque[T]) MarshalJSON() ([]byte, error) {
	return d.ToJSON()
}

// String implements fmt.Stringer.
func (d *ConcurrentDeque[T]) String() string {
	data, err := d.ToJSON()
	return common.JSONResultString(data, err, "[]")
}

// ToJSON serializes ring-buffer values to JSON.
func (r *RingBuffer[T]) ToJSON() ([]byte, error) {
	return json.Marshal(r.Values())
}

// MarshalJSON implements json.Marshaler.
func (r *RingBuffer[T]) MarshalJSON() ([]byte, error) {
	return r.ToJSON()
}

// String implements fmt.Stringer.
func (r *RingBuffer[T]) String() string {
	data, err := r.ToJSON()
	return common.JSONResultString(data, err, "[]")
}

// ToJSON serializes concurrent-ring-buffer values to JSON.
func (r *ConcurrentRingBuffer[T]) ToJSON() ([]byte, error) {
	return json.Marshal(r.Values())
}

// MarshalJSON implements json.Marshaler.
func (r *ConcurrentRingBuffer[T]) MarshalJSON() ([]byte, error) {
	return r.ToJSON()
}

// String implements fmt.Stringer.
func (r *ConcurrentRingBuffer[T]) String() string {
	data, err := r.ToJSON()
	return common.JSONResultString(data, err, "[]")
}

// ToJSON serializes priority queue values to JSON in sorted priority order.
func (pq *PriorityQueue[T]) ToJSON() ([]byte, error) {
	return json.Marshal(pq.ValuesSorted())
}

// MarshalJSON implements json.Marshaler.
func (pq *PriorityQueue[T]) MarshalJSON() ([]byte, error) {
	return pq.ToJSON()
}

// String implements fmt.Stringer.
func (pq *PriorityQueue[T]) String() string {
	data, err := pq.ToJSON()
	return common.JSONResultString(data, err, "[]")
}
