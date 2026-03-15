package eventx

import (
	"reflect"
	"strings"

	"github.com/DaiYuANg/archgo/observabilityx"
)

func (b *Bus) observabilitySafe() observabilityx.Observability {
	if b == nil {
		return observabilityx.Nop()
	}
	return observabilityx.Normalize(b.observability, b.logger)
}

func eventName(event Event) string {
	if event == nil {
		return ""
	}

	name := strings.TrimSpace(event.Name())
	if name != "" {
		return name
	}
	return reflect.TypeOf(event).String()
}
