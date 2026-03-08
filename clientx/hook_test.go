package clientx

import "testing"

func TestHookFuncsDispatch(t *testing.T) {
	var dialCalled bool
	var ioCalled bool

	h := HookFuncs{
		OnDialFunc: func(event DialEvent) {
			dialCalled = event.Protocol == ProtocolTCP
		},
		OnIOFunc: func(event IOEvent) {
			ioCalled = event.Protocol == ProtocolHTTP
		},
	}

	EmitDial([]Hook{h}, DialEvent{Protocol: ProtocolTCP})
	EmitIO([]Hook{h}, IOEvent{Protocol: ProtocolHTTP})

	if !dialCalled {
		t.Fatal("expected dial hook to be called")
	}
	if !ioCalled {
		t.Fatal("expected io hook to be called")
	}
}
