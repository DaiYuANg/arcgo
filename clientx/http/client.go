package http

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/DaiYuANg/archgo/clientx"
	"github.com/samber/lo"
	"resty.dev/v3"
)

type DefaultClient struct {
	raw      *resty.Client
	baseURL  string
	hooks    []clientx.Hook
	policies []clientx.Policy
}

func New(cfg Config, opts ...Option) (Client, error) {
	normalized, err := cfg.NormalizeAndValidate()
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{}
	if normalized.TLS.Enabled || normalized.TLS.InsecureSkipVerify || normalized.TLS.ServerName != "" {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: normalized.TLS.InsecureSkipVerify,
			ServerName:         normalized.TLS.ServerName,
		}
	}

	c := resty.New().
		SetBaseURL(normalized.BaseURL).
		SetTimeout(normalized.Timeout).
		SetTransport(transport)

	if normalized.UserAgent != "" {
		c.SetHeader("User-Agent", normalized.UserAgent)
	}
	if normalized.Headers != nil {
		lo.ForEach(lo.Entries(normalized.Headers.All()), func(entry lo.Entry[string, string], _ int) {
			c.SetHeader(entry.Key, entry.Value)
		})
	}

	client := &DefaultClient{raw: c, baseURL: normalized.BaseURL}
	if normalized.Retry.Enabled {
		client.policies = append(client.policies, clientx.NewRetryPolicy(clientx.RetryPolicyConfig{
			MaxAttempts: max(1, normalized.Retry.MaxRetries+1),
			BaseDelay:   normalized.Retry.WaitMin,
			MaxDelay:    normalized.Retry.WaitMax,
		}))
	}

	clientx.Apply(client, opts...)
	return client, nil
}

func (c *DefaultClient) Close() error {
	if c == nil || c.raw == nil {
		return nil
	}
	if raw := c.raw.Client(); raw != nil {
		raw.CloseIdleConnections()
	}
	return nil
}

func (c *DefaultClient) Raw() *resty.Client {
	return c.raw
}

func (c *DefaultClient) R() *resty.Request {
	return c.raw.R()
}

func (c *DefaultClient) Execute(ctx context.Context, req *resty.Request, method, endpoint string) (*resty.Response, error) {
	op := strings.ToLower(strings.TrimSpace(method))
	if op == "" {
		op = "request"
	}
	addr := c.resolveAddr(endpoint)
	operation := clientx.Operation{
		Protocol: clientx.ProtocolHTTP,
		Kind:     clientx.OperationKindRequest,
		Op:       op,
		Network:  "http",
		Addr:     addr,
	}

	return clientx.InvokeWithPolicies(ctx, operation, func(execCtx context.Context) (*resty.Response, error) {
		workingReq := req
		if workingReq == nil {
			workingReq = c.R()
		}
		workingReq.SetContext(execCtx)

		start := time.Now()
		resp, err := workingReq.Execute(method, endpoint)
		if err != nil {
			wrappedErr := clientx.WrapError(clientx.ProtocolHTTP, op, addr, err)
			clientx.EmitIO(c.hooks, clientx.IOEvent{
				Protocol: clientx.ProtocolHTTP,
				Op:       op,
				Addr:     addr,
				Duration: time.Since(start),
				Err:      wrappedErr,
			})
			return nil, wrappedErr
		}
		clientx.EmitIO(c.hooks, clientx.IOEvent{
			Protocol: clientx.ProtocolHTTP,
			Op:       op,
			Addr:     addr,
			Bytes:    max(0, int(resp.Size())),
			Duration: time.Since(start),
		})
		return resp, nil
	}, c.policies...)
}

func (c *DefaultClient) resolveAddr(endpoint string) string {
	addr := strings.TrimSpace(endpoint)
	if addr == "" {
		return c.baseURL
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") || c.baseURL == "" {
		return addr
	}
	base := strings.TrimRight(c.baseURL, "/")
	if strings.HasPrefix(addr, "/") {
		return base + addr
	}
	return base + "/" + addr
}
