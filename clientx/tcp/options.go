package tcp

import (
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	"github.com/samber/lo"
)

// Option configures a DefaultClient.
type Option func(*DefaultClient)

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
