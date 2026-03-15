package httpapp

import (
	"net/http"
	"testing"

	modelresult "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/resultx"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/require"
)

func TestResultEnvelopeTransformer_KeepEnvelope(t *testing.T) {
	in := modelresult.Result[string]{
		Code:    0,
		Message: "ok",
		Data:    "value",
	}

	out, err := ResultEnvelopeTransformer(nil, "200", in)
	require.NoError(t, err)
	require.Equal(t, in, out)
}

func TestResultEnvelopeTransformer_WrapErrorModel(t *testing.T) {
	in := &huma.ErrorModel{
		Status: http.StatusUnauthorized,
		Detail: "invalid token",
	}

	out, err := ResultEnvelopeTransformer(nil, "401", in)
	require.NoError(t, err)

	wrapped, ok := out.(modelresult.Result[any])
	require.True(t, ok)
	require.Equal(t, http.StatusUnauthorized, wrapped.Code)
	require.Equal(t, "invalid token", wrapped.Message)
	require.Nil(t, wrapped.Data)
}

func TestResultEnvelopeTransformer_KeepNonErrorNonEnvelope(t *testing.T) {
	in := map[string]string{"hello": "world"}

	out, err := ResultEnvelopeTransformer(nil, "200", in)
	require.NoError(t, err)
	require.Equal(t, in, out)
}
