// Package options provides package-level APIs.
package options

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/pkg/option"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

// ServerOptions collects higher-level server construction settings.
type ServerOptions struct {
	Adapter            adapter.Host
	Logger             *slog.Logger
	BasePath           string
	PrintRoutes        bool
	EnableValidation   bool
	Validator          *validator.Validate
	HumaTitle          string
	HumaVersion        string
	HumaDescription    string
	EnablePanicRecover bool
	EnableAccessLog    bool
}

// DefaultServerOptions provides default behavior.
func DefaultServerOptions() *ServerOptions {
	return &ServerOptions{
		Logger:             slog.Default(),
		PrintRoutes:        false,
		HumaTitle:          "My API",
		HumaVersion:        "1.0.0",
		HumaDescription:    "API Documentation",
		EnablePanicRecover: true,
		EnableAccessLog:    false,
	}
}

// ServerOption mutates `ServerOptions`.
type ServerOption func(*ServerOptions)

// Compose combines multiple option functions into one.
func Compose(opts ...ServerOption) ServerOption {
	return func(o *ServerOptions) {
		option.Apply(o, opts...)
	}
}

// WithAdapter configures related behavior.
func WithAdapter(host adapter.Host) ServerOption {
	return func(o *ServerOptions) {
		o.Adapter = host
	}
}

// WithLogger configures related behavior.
func WithLogger(logger *slog.Logger) ServerOption {
	return func(o *ServerOptions) {
		o.Logger = logger
	}
}

// WithBasePath configures related behavior.
func WithBasePath(path string) ServerOption {
	return func(o *ServerOptions) {
		o.BasePath = path
	}
}

// WithPrintRoutes configures related behavior.
func WithPrintRoutes(enabled bool) ServerOption {
	return func(o *ServerOptions) {
		o.PrintRoutes = enabled
	}
}

// WithValidation configures related behavior.
func WithValidation(enabled bool) ServerOption {
	return func(o *ServerOptions) {
		o.EnableValidation = enabled
	}
}

// WithValidator configures related behavior.
func WithValidator(v *validator.Validate) ServerOption {
	return func(o *ServerOptions) {
		o.Validator = v
	}
}

// WithOpenAPIInfo sets OpenAPI title, version, and description fields.
func WithOpenAPIInfo(title, version, description string) ServerOption {
	return func(o *ServerOptions) {
		o.HumaTitle = title
		o.HumaVersion = version
		o.HumaDescription = description
	}
}

// WithPanicRecover enables or disables panic recovery for typed httpx handlers.
func WithPanicRecover(enabled bool) ServerOption {
	return func(o *ServerOptions) {
		o.EnablePanicRecover = enabled
	}
}

// WithAccessLog enables or disables request logging in the httpx layer.
func WithAccessLog(enabled bool) ServerOption {
	return func(o *ServerOptions) {
		o.EnableAccessLog = enabled
	}
}

// Build converts `ServerOptions` into `httpx.ServerOption` values.
func (o *ServerOptions) Build() collectionx.List[httpx.ServerOption] {
	opts := collectionx.NewList[httpx.ServerOption](
		httpx.WithLogger(o.Logger),
		httpx.WithPrintRoutes(o.PrintRoutes),
	)

	if o.Adapter != nil {
		opts.Add(httpx.WithAdapter(o.Adapter))
	}
	if o.BasePath != "" {
		opts.Add(httpx.WithBasePath(o.BasePath))
	}
	appendValidationBuildOption(opts, o)
	opts.Add(
		httpx.WithOpenAPIInfo(o.HumaTitle, o.HumaVersion, o.HumaDescription),
		httpx.WithPanicRecover(o.EnablePanicRecover),
		httpx.WithAccessLog(o.EnableAccessLog),
	)

	return opts
}

// HTTPClientOptions collects standard `http.Client` construction settings.
type HTTPClientOptions struct {
	Timeout   time.Duration
	Transport http.RoundTripper
	Jar       http.CookieJar
}

// DefaultHTTPClientOptions provides default behavior.
func DefaultHTTPClientOptions() *HTTPClientOptions {
	return &HTTPClientOptions{
		Timeout: 30 * time.Second,
	}
}

// HTTPClientOption mutates `HTTPClientOptions`.
type HTTPClientOption func(*HTTPClientOptions)

// WithHTTPTimeout configures related behavior.
func WithHTTPTimeout(timeout time.Duration) HTTPClientOption {
	return func(o *HTTPClientOptions) {
		o.Timeout = timeout
	}
}

// WithHTTPTransport configures related behavior.
func WithHTTPTransport(transport http.RoundTripper) HTTPClientOption {
	return func(o *HTTPClientOptions) {
		o.Transport = transport
	}
}

// WithHTTPCookieJar configures related behavior.
func WithHTTPCookieJar(jar http.CookieJar) HTTPClientOption {
	return func(o *HTTPClientOptions) {
		o.Jar = jar
	}
}

// Build constructs an `http.Client` from the configured options.
func (o *HTTPClientOptions) Build() *http.Client {
	return &http.Client{
		Timeout:   o.Timeout,
		Transport: o.Transport,
		Jar:       o.Jar,
	}
}

// ContextOptions collects helper settings for building a context.Context.
type ContextOptions struct {
	Timeout   time.Duration
	Deadline  time.Time
	ValueKeys map[contextValueKey]any
}

type contextValueKey string

// ContextOption mutates `ContextOptions`.
type ContextOption func(*ContextOptions)

// WithContextTimeout configures related behavior.
func WithContextTimeout(timeout time.Duration) ContextOption {
	return func(o *ContextOptions) {
		o.Timeout = timeout
	}
}

// WithContextDeadline configures related behavior.
func WithContextDeadline(deadline time.Time) ContextOption {
	return func(o *ContextOptions) {
		o.Deadline = deadline
	}
}

// WithContextValue configures related behavior.
func WithContextValue(key string, value any) ContextOption {
	return func(o *ContextOptions) {
		ensureContextValues(o)[contextValueKey(key)] = value
	}
}

// Build creates a context and optional cancel function from the configured values.
func (o *ContextOptions) Build() (context.Context, context.CancelFunc) {
	ctx, cancel := baseContext(o)
	return applyContextValues(ctx, o.ValueKeys), cancel
}

// WithContextValueOpt mutates a ContextOptions value directly.
func WithContextValueOpt(o *ContextOptions, key string, value any) *ContextOptions {
	ensureContextValues(o)[contextValueKey(key)] = value
	return o
}

func appendValidationBuildOption(opts collectionx.List[httpx.ServerOption], o *ServerOptions) {
	if o.Validator != nil {
		opts.Add(httpx.WithValidator(o.Validator))
		return
	}
	if o.EnableValidation {
		opts.Add(httpx.WithValidation())
	}
}

func ensureContextValues(o *ContextOptions) map[contextValueKey]any {
	if o.ValueKeys == nil {
		o.ValueKeys = make(map[contextValueKey]any)
	}
	return o.ValueKeys
}

func baseContext(o *ContextOptions) (context.Context, context.CancelFunc) {
	switch {
	case !o.Deadline.IsZero():
		return context.WithDeadline(context.Background(), o.Deadline)
	case o.Timeout > 0:
		return context.WithTimeout(context.Background(), o.Timeout)
	default:
		return context.Background(), nil
	}
}

func applyContextValues(ctx context.Context, values map[contextValueKey]any) context.Context {
	if len(values) == 0 {
		return ctx
	}

	return lo.Reduce(lo.Keys(values), func(current context.Context, key contextValueKey, _ int) context.Context {
		return context.WithValue(current, key, values[key])
	}, ctx)
}
