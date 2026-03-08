package http

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	"github.com/samber/lo"
	"resty.dev/v3"
)

type DefaultClient struct {
	raw     *resty.Client
	baseURL string
	hooks   []clientx.Hook
}

func New(cfg Config, opts ...Option) Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
			ServerName:         cfg.TLS.ServerName,
		},
	}

	c := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetTimeout(cfg.Timeout).
		SetTransport(transport)

	if cfg.UserAgent != "" {
		c.SetHeader("User-Agent", cfg.UserAgent)
	}
	for k, v := range cfg.Headers.All() {
		c.SetHeader(k, v)
	}

	if cfg.Retry.Enabled {
		c.SetRetryCount(cfg.Retry.MaxRetries)
		c.SetRetryWaitTime(cfg.Retry.WaitMin)
		c.SetRetryMaxWaitTime(cfg.Retry.WaitMax)
	}

	client := &DefaultClient{raw: c, baseURL: cfg.BaseURL}
	lo.ForEach(opts, func(opt Option, index int) {
		opt(client)
	})
	var api Client = client
	return api
}

func (c *DefaultClient) Raw() *resty.Client {
	return c.raw
}

func (c *DefaultClient) R() *resty.Request {
	return c.raw.R()
}

func (c *DefaultClient) Execute(req *resty.Request, method, endpoint string) (*resty.Response, error) {
	if req == nil {
		req = c.R()
	}
	start := time.Now()
	addr := c.resolveAddr(endpoint)
	op := strings.ToLower(strings.TrimSpace(method))
	if op == "" {
		op = "request"
	}

	resp, err := req.Execute(method, endpoint)
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
