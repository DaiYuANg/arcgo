package http

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/DaiYuANg/archgo/clientx"
	"github.com/DaiYuANg/archgo/collectionx"
)

type Config struct {
	BaseURL   string
	Timeout   time.Duration
	Headers   collectionx.Map[string, string]
	UserAgent string
	Retry     clientx.RetryConfig
	TLS       clientx.TLSConfig
}

const defaultTimeout = 30 * time.Second

var ErrInvalidConfig = errors.New("invalid http client config")

func (cfg Config) NormalizeAndValidate() (Config, error) {
	out := cfg
	out.BaseURL = strings.TrimSpace(out.BaseURL)
	out.UserAgent = strings.TrimSpace(out.UserAgent)

	if out.BaseURL != "" {
		if _, err := url.ParseRequestURI(out.BaseURL); err != nil {
			return Config{}, fmt.Errorf("%w: base_url: %v", ErrInvalidConfig, err)
		}
	}

	if out.Timeout == 0 {
		out.Timeout = defaultTimeout
	}
	if out.Timeout < 0 {
		return Config{}, fmt.Errorf("%w: timeout must be >= 0", ErrInvalidConfig)
	}

	if out.Retry.MaxRetries < 0 {
		return Config{}, fmt.Errorf("%w: retry.max_retries must be >= 0", ErrInvalidConfig)
	}
	if out.Retry.WaitMin < 0 || out.Retry.WaitMax < 0 {
		return Config{}, fmt.Errorf("%w: retry wait durations must be >= 0", ErrInvalidConfig)
	}
	if out.Retry.WaitMax > 0 && out.Retry.WaitMin > out.Retry.WaitMax {
		return Config{}, fmt.Errorf("%w: retry.wait_min must be <= retry.wait_max", ErrInvalidConfig)
	}

	return out, nil
}
