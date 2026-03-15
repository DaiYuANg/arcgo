package preset

import (
	"time"

	clientudp "github.com/DaiYuANg/archgo/clientx/udp"
)

type lowLatencyUDPPreset struct {
	dialTimeout      time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
	timeoutGuard     time.Duration
	concurrencyLimit int
	options          []clientudp.Option
}

type LowLatencyUDPOption func(*lowLatencyUDPPreset)

func WithLowLatencyUDPDialTimeout(timeout time.Duration) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.dialTimeout = timeout
	}
}

func WithLowLatencyUDPReadTimeout(timeout time.Duration) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.readTimeout = timeout
	}
}

func WithLowLatencyUDPWriteTimeout(timeout time.Duration) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.writeTimeout = timeout
	}
}

func WithLowLatencyUDPTimeoutGuard(timeout time.Duration) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.timeoutGuard = timeout
	}
}

func WithLowLatencyUDPConcurrencyLimit(maxInFlight int) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		p.concurrencyLimit = maxInFlight
	}
}

func WithLowLatencyUDPOption(opt clientudp.Option) LowLatencyUDPOption {
	return func(p *lowLatencyUDPPreset) {
		if opt != nil {
			p.options = append(p.options, opt)
		}
	}
}

func NewLowLatencyUDP(cfg clientudp.Config, opts ...LowLatencyUDPOption) (clientudp.Client, error) {
	preset := defaultLowLatencyUDPPreset()
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

	clientOpts := make([]clientudp.Option, 0, 2+len(preset.options))
	if preset.timeoutGuard > 0 {
		clientOpts = append(clientOpts, clientudp.WithTimeoutGuard(preset.timeoutGuard))
	}
	if preset.concurrencyLimit > 0 {
		clientOpts = append(clientOpts, clientudp.WithConcurrencyLimit(preset.concurrencyLimit))
	}
	clientOpts = append(clientOpts, preset.options...)
	return clientudp.New(tuned, clientOpts...)
}

func defaultLowLatencyUDPPreset() lowLatencyUDPPreset {
	return lowLatencyUDPPreset{
		dialTimeout:      300 * time.Millisecond,
		readTimeout:      150 * time.Millisecond,
		writeTimeout:     150 * time.Millisecond,
		timeoutGuard:     200 * time.Millisecond,
		concurrencyLimit: 1024,
	}
}
