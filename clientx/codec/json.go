package codec

import "encoding/json"

type jsonCodec struct{}

func (c jsonCodec) Name() string {
	return "json"
}

func (c jsonCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (c jsonCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

var JSON Codec = jsonCodec{}
