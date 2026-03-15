package http

import (
	"context"

	"github.com/DaiYuANg/archgo/clientx"
	"resty.dev/v3"
)

type Client interface {
	clientx.Closer
	Raw() *resty.Client
	R() *resty.Request
	Execute(ctx context.Context, req *resty.Request, method, endpoint string) (*resty.Response, error)
}

var _ Client = (*DefaultClient)(nil)
