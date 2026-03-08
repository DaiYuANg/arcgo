package http

import "resty.dev/v3"

type Client interface {
	Raw() *resty.Client
	R() *resty.Request
	Execute(req *resty.Request, method, endpoint string) (*resty.Response, error)
}

var _ Client = (*DefaultClient)(nil)
