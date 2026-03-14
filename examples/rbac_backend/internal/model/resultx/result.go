package resultx

const (
	CodeOK      = 0
	MessageOK   = "ok"
	MessageFail = "fail"
)

type Result[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func (Result[T]) IsResultEnvelope() bool {
	return true
}
