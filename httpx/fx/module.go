package fx

import (
	"github.com/DaiYuANg/arcgo/httpx"
	"go.uber.org/fx"
)

func NewHttpxModule(options ...httpx.ServerOption) {
	fx.Module("httpx", fx.Provide(
		httpx.New(options...),
	))
}
