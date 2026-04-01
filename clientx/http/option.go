package http

import (
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	"github.com/samber/lo"
	"resty.dev/v3"
)

// Option configures a DefaultClient.
type Option func(*DefaultClient)

// WithRequestMiddleware adds a resty request middleware.
func WithRequestMiddleware(fn func(*resty.Client, *resty.Request) error) Option {
	return func(c *DefaultClient) {
		c.Raw().AddRequestMiddleware(fn)
	}
}

// WithResponseMiddleware adds a resty response middleware.
func WithResponseMiddleware(fn func(*resty.Client, *resty.Response) error) Option {
	return func(c *DefaultClient) {
		c.Raw().AddResponseMiddleware(fn)
	}
}

// WithHeader adds a default request header.
func WithHeader(key, value string) Option {
	return func(c *DefaultClient) {
		c.Raw().SetHeader(key, value)
	}
}

// WithHooks appends client hooks.
func WithHooks(hooks ...clientx.Hook) Option {
	filtered := lo.Filter(hooks, func(h clientx.Hook, _ int) bool {
		return h != nil
	})
	return func(c *DefaultClient) {
		c.hooks = lo.Concat(c.hooks, filtered)
	}
}

// WithPolicies appends execution policies.
func WithPolicies(policies ...clientx.Policy) Option {
	filtered := lo.Filter(policies, func(p clientx.Policy, _ int) bool {
		return p != nil
	})
	return func(c *DefaultClient) {
		c.policies = lo.Concat(c.policies, filtered)
	}
}

// WithConcurrencyLimit adds a concurrency limit policy.
func WithConcurrencyLimit(maxInFlight int) Option {
	return func(c *DefaultClient) {
		c.policies = lo.Concat(c.policies, []clientx.Policy{clientx.NewConcurrencyLimitPolicy(maxInFlight)})
	}
}

// WithTimeoutGuard adds a timeout guard policy.
func WithTimeoutGuard(timeout time.Duration) Option {
	return func(c *DefaultClient) {
		c.policies = lo.Concat(c.policies, []clientx.Policy{clientx.NewTimeoutPolicy(timeout)})
	}
}
