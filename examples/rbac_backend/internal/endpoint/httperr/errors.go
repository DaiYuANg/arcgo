package httperr

import "github.com/DaiYuANg/archgo/httpx"

func Unauthorized(message string) error {
	return httpx.NewError(401, message)
}

func NotFound(message string) error {
	return httpx.NewError(404, message)
}

func BadRequest(message string) error {
	return httpx.NewError(400, message)
}
