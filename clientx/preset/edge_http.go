package preset

import (
	"strings"
	"time"

	"github.com/DaiYuANg/archgo/clientx"
	clienthttp "github.com/DaiYuANg/archgo/clientx/http"
)

type edgeHTTPPreset struct {
	timeout          time.Duration
	timeoutGuard     time.Duration
	concurrencyLimit int
	retry            clientx.RetryConfig
	userAgent        string
	disableRetry     bool
	options          []clienthttp.Option
}

type EdgeHTTPOption func(*edgeHTTPPreset)

func WithEdgeHTTPTimeout(timeout time.Duration) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.timeout = timeout
	}
}

func WithEdgeHTTPTimeoutGuard(timeout time.Duration) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.timeoutGuard = timeout
	}
}

func WithEdgeHTTPConcurrencyLimit(maxInFlight int) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.concurrencyLimit = maxInFlight
	}
}

func WithEdgeHTTPRetry(cfg clientx.RetryConfig) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.retry = cfg
		p.disableRetry = false
	}
}

func WithEdgeHTTPDisableRetry() EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.disableRetry = true
	}
}

func WithEdgeHTTPUserAgent(userAgent string) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		p.userAgent = strings.TrimSpace(userAgent)
	}
}

func WithEdgeHTTPOption(opt clienthttp.Option) EdgeHTTPOption {
	return func(p *edgeHTTPPreset) {
		if opt != nil {
			p.options = append(p.options, opt)
		}
	}
}

func NewEdgeHTTP(cfg clienthttp.Config, opts ...EdgeHTTPOption) (clienthttp.Client, error) {
	preset := defaultEdgeHTTPPreset()
	for _, opt := range opts {
		if opt != nil {
			opt(&preset)
		}
	}

	tuned := cfg
	if tuned.Timeout == 0 {
		tuned.Timeout = preset.timeout
	}
	if strings.TrimSpace(tuned.UserAgent) == "" && preset.userAgent != "" {
		tuned.UserAgent = preset.userAgent
	}
	if !preset.disableRetry {
		if isZeroRetryConfig(tuned.Retry) {
			tuned.Retry = preset.retry
		}
		if !tuned.Retry.Enabled && hasRetryHint(tuned.Retry) {
			tuned.Retry.Enabled = true
		}
	}

	clientOpts := make([]clienthttp.Option, 0, 2+len(preset.options))
	if preset.timeoutGuard > 0 {
		clientOpts = append(clientOpts, clienthttp.WithTimeoutGuard(preset.timeoutGuard))
	}
	if preset.concurrencyLimit > 0 {
		clientOpts = append(clientOpts, clienthttp.WithConcurrencyLimit(preset.concurrencyLimit))
	}
	clientOpts = append(clientOpts, preset.options...)
	return clienthttp.New(tuned, clientOpts...)
}

func defaultEdgeHTTPPreset() edgeHTTPPreset {
	return edgeHTTPPreset{
		timeout:          5 * time.Second,
		timeoutGuard:     4 * time.Second,
		concurrencyLimit: 256,
		retry: clientx.RetryConfig{
			Enabled:    true,
			MaxRetries: 2,
			WaitMin:    50 * time.Millisecond,
			WaitMax:    250 * time.Millisecond,
		},
		userAgent: "arcgo-clientx/edge-http",
	}
}
