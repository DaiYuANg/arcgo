package codec

import "fmt"

type bytesCodec struct{}

func (c bytesCodec) Name() string {
	return "bytes"
}

func (c bytesCodec) Marshal(v any) ([]byte, error) {
	switch value := v.(type) {
	case []byte:
		return append([]byte(nil), value...), nil
	default:
		return nil, fmt.Errorf("%w: codec=bytes marshal %T", ErrUnsupportedValue, v)
	}
}

func (c bytesCodec) Unmarshal(data []byte, v any) error {
	target, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("%w: codec=bytes unmarshal %T", ErrUnsupportedValue, v)
	}
	*target = append((*target)[:0], data...)
	return nil
}

var Bytes Codec = bytesCodec{}
