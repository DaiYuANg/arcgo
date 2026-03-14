package response

import modelresult "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/model/resultx"

func OK[T any](data T) *modelresult.Result[T] {
	return &modelresult.Result[T]{
		Code:    modelresult.CodeOK,
		Message: modelresult.MessageOK,
		Data:    data,
	}
}
