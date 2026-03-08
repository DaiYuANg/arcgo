package clientx

import (
	"time"

	"github.com/samber/lo"
)

type Hook interface {
	OnDial(event DialEvent)
	OnIO(event IOEvent)
}

type HookFuncs struct {
	OnDialFunc func(event DialEvent)
	OnIOFunc   func(event IOEvent)
}

func (h HookFuncs) OnDial(event DialEvent) {
	if h.OnDialFunc != nil {
		h.OnDialFunc(event)
	}
}

func (h HookFuncs) OnIO(event IOEvent) {
	if h.OnIOFunc != nil {
		h.OnIOFunc(event)
	}
}

type DialEvent struct {
	Protocol Protocol
	Op       string
	Network  string
	Addr     string
	Duration time.Duration
	Err      error
}

type IOEvent struct {
	Protocol Protocol
	Op       string
	Addr     string
	Bytes    int
	Duration time.Duration
	Err      error
}

func EmitDial(hooks []Hook, event DialEvent) {
	lo.ForEach(hooks, func(h Hook, _ int) {
		if h != nil {
			h.OnDial(event)
		}
	})
}

func EmitIO(hooks []Hook, event IOEvent) {
	lo.ForEach(hooks, func(h Hook, _ int) {
		if h != nil {
			h.OnIO(event)
		}
	})
}
