package udp

import "github.com/DaiYuANg/arcgo/clientx"

type Option func(*DefaultClient)

func WithHooks(hooks ...clientx.Hook) Option {
	return func(c *DefaultClient) {
		c.hooks = append(c.hooks, hooks...)
	}
}
