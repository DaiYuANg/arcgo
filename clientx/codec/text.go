package codec

import (
	"encoding"
	"fmt"
)

type textCodec struct{}

func (c textCodec) Name() string {
	return "text"
}

func (c textCodec) Marshal(v any) ([]byte, error) {
	switch value := v.(type) {
	case string:
		return []byte(value), nil
	case []byte:
		return append([]byte(nil), value...), nil
	case encoding.TextMarshaler:
		return value.MarshalText()
	case fmt.Stringer:
		return []byte(value.String()), nil
	default:
		return nil, fmt.Errorf("%w: codec=text marshal %T", ErrUnsupportedValue, v)
	}
}

func (c textCodec) Unmarshal(data []byte, v any) error {
	switch target := v.(type) {
	case *string:
		*target = string(data)
		return nil
	case *[]byte:
		*target = append((*target)[:0], data...)
		return nil
	case encoding.TextUnmarshaler:
		return target.UnmarshalText(data)
	default:
		return fmt.Errorf("%w: codec=text unmarshal %T", ErrUnsupportedValue, v)
	}
}

var Text Codec = textCodec{}
