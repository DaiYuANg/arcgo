package http

import (
	"github.com/DaiYuANg/arcgo/clientx"
	"resty.dev/v3"
)

type Option func(*DefaultClient)

func WithRequestMiddleware(fn func(*resty.Client, *resty.Request) error) Option {
	return func(c *DefaultClient) {
		c.Raw().AddRequestMiddleware(fn)
	}
}

func WithResponseMiddleware(fn func(*resty.Client, *resty.Response) error) Option {
	return func(c *DefaultClient) {
		c.Raw().AddResponseMiddleware(fn)
	}
}

func WithHeader(key, value string) Option {
	return func(c *DefaultClient) {
		c.Raw().SetHeader(key, value)
	}
}

func WithHooks(hooks ...clientx.Hook) Option {
	return func(c *DefaultClient) {
		c.hooks = append(c.hooks, hooks...)
	}
}
