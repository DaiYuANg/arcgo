package codec

import (
	"bytes"
	"errors"
	"testing"
)

type sample struct {
	Name string `json:"name"`
}

type reverseCodec struct{}

func (c reverseCodec) Name() string {
	return "reverse"
}

func (c reverseCodec) Marshal(v any) ([]byte, error) {
	s, ok := v.(string)
	if !ok {
		return nil, ErrUnsupportedValue
	}
	raw := []byte(s)
	for i, j := 0, len(raw)-1; i < j; i, j = i+1, j-1 {
		raw[i], raw[j] = raw[j], raw[i]
	}
	return raw, nil
}

func (c reverseCodec) Unmarshal(data []byte, v any) error {
	target, ok := v.(*string)
	if !ok {
		return ErrUnsupportedValue
	}
	raw := append([]byte(nil), data...)
	for i, j := 0, len(raw)-1; i < j; i, j = i+1, j-1 {
		raw[i], raw[j] = raw[j], raw[i]
	}
	*target = string(raw)
	return nil
}

func TestDefaultRegistryBuiltins(t *testing.T) {
	for _, name := range []string{"bytes", "json", "text"} {
		if _, ok := Get(name); !ok {
			t.Fatalf("expected builtin codec %q", name)
		}
	}
	if GetOption("json").IsAbsent() {
		t.Fatal("expected json codec option to be present")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	raw, err := JSON.Marshal(sample{Name: "arc"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var out sample
	if err := JSON.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if out.Name != "arc" {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

func TestTextRoundTrip(t *testing.T) {
	raw, err := Text.Marshal("hello")
	if err != nil {
		t.Fatalf("marshal text failed: %v", err)
	}
	var out string
	if err := Text.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal text failed: %v", err)
	}
	if out != "hello" {
		t.Fatalf("unexpected decoded text: %q", out)
	}
}

func TestBytesUnsupportedType(t *testing.T) {
	_, err := Bytes.Marshal("not-bytes")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrUnsupportedValue) {
		t.Fatalf("expected ErrUnsupportedValue, got %v", err)
	}
}

func TestLengthPrefixedRoundTrip(t *testing.T) {
	f := NewLengthPrefixed(32)
	buf := bytes.NewBuffer(nil)

	if err := f.WriteFrame(buf, []byte("ping")); err != nil {
		t.Fatalf("write frame failed: %v", err)
	}
	raw, err := f.ReadFrame(buf)
	if err != nil {
		t.Fatalf("read frame failed: %v", err)
	}
	if string(raw) != "ping" {
		t.Fatalf("unexpected frame: %q", raw)
	}
}

func TestLengthPrefixedExceedLimit(t *testing.T) {
	f := NewLengthPrefixed(2)
	err := f.WriteFrame(bytes.NewBuffer(nil), []byte("hello"))
	if err == nil {
		t.Fatal("expected frame size error, got nil")
	}
}

func TestRegisterCustomCodec(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(reverseCodec{}); err != nil {
		t.Fatalf("register custom codec failed: %v", err)
	}

	c, ok := r.Get("reverse")
	if !ok {
		t.Fatal("expected registered codec")
	}

	raw, err := c.Marshal("abc")
	if err != nil {
		t.Fatalf("marshal reverse failed: %v", err)
	}
	if string(raw) != "cba" {
		t.Fatalf("unexpected marshaled value: %q", raw)
	}

	var out string
	if err := c.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal reverse failed: %v", err)
	}
	if out != "abc" {
		t.Fatalf("unexpected decoded value: %q", out)
	}
}
