package httpapp

import (
	"net/http"
	"strconv"
	"strings"

	modelresult "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/resultx"
	"github.com/danielgtaylor/huma/v2"
)

type resultEnvelope interface {
	IsResultEnvelope() bool
}

func ResultEnvelopeTransformer(_ huma.Context, status string, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	if envelope, ok := v.(resultEnvelope); ok && envelope.IsResultEnvelope() {
		return v, nil
	}

	code := parseStatusCode(status)
	if code < http.StatusBadRequest {
		return v, nil
	}

	return modelresult.Result[any]{
		Code:    code,
		Message: errorMessage(code, v),
		Data:    nil,
	}, nil
}

func parseStatusCode(status string) int {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		return http.StatusInternalServerError
	}

	if code, err := strconv.Atoi(trimmed); err == nil {
		return code
	}

	if idx := strings.IndexByte(trimmed, ' '); idx > 0 {
		if code, err := strconv.Atoi(trimmed[:idx]); err == nil {
			return code
		}
	}

	return http.StatusInternalServerError
}

func errorMessage(code int, v any) string {
	switch errModel := v.(type) {
	case *huma.ErrorModel:
		return messageFromErrorModel(errModel, code)
	case huma.ErrorModel:
		return messageFromErrorModel(&errModel, code)
	case error:
		if msg := strings.TrimSpace(errModel.Error()); msg != "" {
			return msg
		}
	}
	return http.StatusText(code)
}

func messageFromErrorModel(errModel *huma.ErrorModel, code int) string {
	if errModel == nil {
		return http.StatusText(code)
	}
	if msg := strings.TrimSpace(errModel.Detail); msg != "" {
		return msg
	}
	if msg := strings.TrimSpace(errModel.Title); msg != "" {
		return msg
	}
	if len(errModel.Errors) > 0 && errModel.Errors[0] != nil {
		if msg := strings.TrimSpace(errModel.Errors[0].Message); msg != "" {
			return msg
		}
	}
	return http.StatusText(code)
}
