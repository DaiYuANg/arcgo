package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseEndpoint_JSONMarshalError(t *testing.T) {
	w := httptest.NewRecorder()
	endpoint := BaseEndpoint{}

	// channel 不能被 JSON 编码
	endpoint.JSON(w, map[string]interface{}{"invalid": make(chan int)}, http.StatusAccepted)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "internal server error")
}

func TestBaseEndpoint_SuccessMarshalError(t *testing.T) {
	w := httptest.NewRecorder()
	endpoint := BaseEndpoint{}

	endpoint.Success(w, map[string]interface{}{"invalid": make(chan int)})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "internal server error")
}

func TestBaseEndpoint_JSONSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	endpoint := BaseEndpoint{}

	endpoint.JSON(w, map[string]string{"message": "ok"}, http.StatusCreated)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "\"message\":\"ok\"")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}
