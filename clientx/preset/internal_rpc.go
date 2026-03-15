package preset

import (
	"time"

	"github.com/DaiYuANg/archgo/clientx"
	clienttcp "github.com/DaiYuANg/archgo/clientx/tcp"
)

type internalRPCPreset struct {
	dialTimeout      time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
	keepAlive        time.Duration
	timeoutGuard     time.Duration
	concurrencyLimit int
	retryPolicy      clientx.RetryPolicyConfig
	disableRetry     bool
	options          []clienttcp.Option
}

type InternalRPCOption func(*internalRPCPreset)

func WithInternalRPCDialTimeout(timeout time.Duration) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.dialTimeout = timeout
	}
}

func WithInternalRPCReadTimeout(timeout time.Duration) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.readTimeout = timeout
	}
}

func WithInternalRPCWriteTimeout(timeout time.Duration) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.writeTimeout = timeout
	}
}

func WithInternalRPCKeepAlive(keepAlive time.Duration) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.keepAlive = keepAlive
	}
}

func WithInternalRPCTimeoutGuard(timeout time.Duration) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.timeoutGuard = timeout
	}
}

func WithInternalRPCConcurrencyLimit(maxInFlight int) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.concurrencyLimit = maxInFlight
	}
}

func WithInternalRPCRetryPolicy(cfg clientx.RetryPolicyConfig) InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.retryPolicy = cfg
		p.disableRetry = false
	}
}

func WithInternalRPCDisableRetry() InternalRPCOption {
	return func(p *internalRPCPreset) {
		p.disableRetry = true
	}
}

func WithInternalRPCOption(opt clienttcp.Option) InternalRPCOption {
	return func(p *internalRPCPreset) {
		if opt != nil {
			p.options = append(p.options, opt)
		}
	}
}

func NewInternalRPC(cfg clienttcp.Config, opts ...InternalRPCOption) (clienttcp.Client, error) {
	preset := defaultInternalRPCPreset()
	for _, opt := range opts {
		if opt != nil {
			opt(&preset)
		}
	}

	tuned := cfg
	if tuned.DialTimeout == 0 {
		tuned.DialTimeout = preset.dialTimeout
	}
	if tuned.ReadTimeout == 0 {
		tuned.ReadTimeout = preset.readTimeout
	}
	if tuned.WriteTimeout == 0 {
		tuned.WriteTimeout = preset.writeTimeout
	}
	if tuned.KeepAlive == 0 {
		tuned.KeepAlive = preset.keepAlive
	}

	clientOpts := make([]clienttcp.Option, 0, 3+len(preset.options))
	if preset.timeoutGuard > 0 {
		clientOpts = append(clientOpts, clienttcp.WithTimeoutGuard(preset.timeoutGuard))
	}
	if preset.concurrencyLimit > 0 {
		clientOpts = append(clientOpts, clienttcp.WithConcurrencyLimit(preset.concurrencyLimit))
	}
	if !preset.disableRetry {
		clientOpts = append(clientOpts, clienttcp.WithPolicies(clientx.NewRetryPolicy(preset.retryPolicy)))
	}
	clientOpts = append(clientOpts, preset.options...)
	return clienttcp.New(tuned, clientOpts...)
}

func defaultInternalRPCPreset() internalRPCPreset {
	return internalRPCPreset{
		dialTimeout:      1500 * time.Millisecond,
		readTimeout:      2 * time.Second,
		writeTimeout:     2 * time.Second,
		keepAlive:        30 * time.Second,
		timeoutGuard:     2 * time.Second,
		concurrencyLimit: 512,
		retryPolicy: clientx.RetryPolicyConfig{
			MaxAttempts: 3,
			BaseDelay:   20 * time.Millisecond,
			MaxDelay:    120 * time.Millisecond,
			Multiplier:  2,
			JitterRatio: 0.2,
		},
	}
}
