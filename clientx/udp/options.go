package udp

import (
	"time"

	"github.com/DaiYuANg/archgo/clientx"
	"github.com/samber/lo"
)

type Option func(*DefaultClient)

func WithHooks(hooks ...clientx.Hook) Option {
	filtered := lo.Filter(hooks, func(h clientx.Hook, _ int) bool {
		return h != nil
	})
	return func(c *DefaultClient) {
		c.hooks = append(c.hooks, filtered...)
	}
}

func WithPolicies(policies ...clientx.Policy) Option {
	filtered := lo.Filter(policies, func(p clientx.Policy, _ int) bool {
		return p != nil
	})
	return func(c *DefaultClient) {
		c.policies = append(c.policies, filtered...)
	}
}

func WithConcurrencyLimit(maxInFlight int) Option {
	return func(c *DefaultClient) {
		c.policies = append(c.policies, clientx.NewConcurrencyLimitPolicy(maxInFlight))
	}
}

func WithTimeoutGuard(timeout time.Duration) Option {
	return func(c *DefaultClient) {
		c.policies = append(c.policies, clientx.NewTimeoutPolicy(timeout))
	}
}
