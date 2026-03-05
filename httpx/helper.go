package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/samber/lo"
)

// BaseEndpoint 基础 endpoint 结构，提供常用辅助方法
type BaseEndpoint struct{}

// JSON 返回 JSON 响应
func (BaseEndpoint) JSON(w http.ResponseWriter, data interface{}, status ...int) {
	code := lo.FirstOr(status, http.StatusOK)
	writeJSON(w, code, data)
}

// Error 返回错误响应
func (BaseEndpoint) Error(w http.ResponseWriter, message string, code ...int) {
	status := lo.FirstOr(code, http.StatusInternalServerError)
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

// Success 返回成功响应
func (BaseEndpoint) Success(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   data,
	})
}

// GetHeader 获取请求头
func (BaseEndpoint) GetHeader(r *http.Request, key string) string {
	return r.Header.Get(key)
}

// GetQuery 获取查询参数
func (BaseEndpoint) GetQuery(r *http.Request, key string, defaultValue ...string) string {
	value := r.URL.Query().Get(key)
	return lo.Ternary(value == "" && len(defaultValue) > 0, defaultValue[0], value)
}

// GetQueryOrDefault 获取查询参数，带默认值
func (BaseEndpoint) GetQueryOrDefault(r *http.Request, key string, defaultValue string) string {
	value := r.URL.Query().Get(key)
	return lo.Ternary(value == "", defaultValue, value)
}

// HandlerEndpoint 实际的处理 endpoint 示例
type HandlerEndpoint struct {
	BaseEndpoint
}

// GetUserList 获取用户列表
// 通过方法名自动生成路由：GET /user/list
func (e *HandlerEndpoint) GetUserList(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"users": []string{"user1", "user2", "user3"},
	})
	return nil
}

// GetUserByID 获取单个用户
// 通过方法名自动生成路由：GET /user/by/id
func (e *HandlerEndpoint) GetUserByID(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := e.GetQuery(r, "id", "1")
	e.Success(w, map[string]interface{}{
		"user": "user" + id,
	})
	return nil
}

// CreateNewUser 创建用户
// 通过方法名自动生成路由：POST /new/user
func (e *HandlerEndpoint) CreateNewUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"message": "user created",
	})
	return nil
}

// UpdateUserInfo 更新用户
// 通过方法名自动生成路由：PUT /user/info
func (e *HandlerEndpoint) UpdateUserInfo(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"message": "user updated",
	})
	return nil
}

// DeleteUserByID 删除用户
// 通过方法名自动生成路由：DELETE /user/by/id
func (e *HandlerEndpoint) DeleteUserByID(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"message": "user deleted",
	})
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Default().Error(
			"Failed to marshal JSON response",
			slog.Int("status", status),
			slog.String("payload_type", typeNameOf(payload)),
			slog.String("error", err.Error()),
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err = w.Write(body); err != nil {
		slog.Default().Error(
			"Failed to write JSON response",
			slog.Int("status", status),
			slog.Int("bytes", len(body)),
			slog.String("error", err.Error()),
		)
		return
	}
}

func typeNameOf(v any) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", v)
}
