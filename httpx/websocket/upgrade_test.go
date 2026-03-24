package websocket

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/lxzan/gws"
)

func TestOpcodeMappings(t *testing.T) {
	cases := []struct {
		name string
		typ  MessageType
		op   gws.Opcode
	}{
		{name: "text", typ: MessageText, op: gws.OpcodeText},
		{name: "binary", typ: MessageBinary, op: gws.OpcodeBinary},
		{name: "ping", typ: MessagePing, op: gws.OpcodePing},
		{name: "pong", typ: MessagePong, op: gws.OpcodePong},
		{name: "close", typ: MessageClose, op: gws.OpcodeCloseConnection},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotOp, ok := toGWSOpcode(tc.typ)
			if !ok || gotOp != tc.op {
				t.Fatalf("toGWSOpcode mismatch: ok=%v op=%v", ok, gotOp)
			}
			gotType, ok := fromGWSOpcode(tc.op)
			if !ok || gotType != tc.typ {
				t.Fatalf("fromGWSOpcode mismatch: ok=%v type=%v", ok, gotType)
			}
		})
	}
}

func TestUpgradeNilHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)
	rec := httptest.NewRecorder()

	err := Upgrade(rec, req, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUpgradeFailed) {
		t.Fatalf("expected ErrUpgradeFailed, got %v", err)
	}
}
