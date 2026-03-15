package shared

import (
	"context"
	"time"

	"github.com/DaiYuANg/archgo/httpx"
	"github.com/danielgtaylor/huma/v2"
)

type ListUsersInput struct {
	Limit int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Page  int    `query:"page" validate:"omitempty,min=1"`
	Q     string `query:"q" validate:"omitempty,max=100"`
}

type ListUsersOutput struct {
	Body struct {
		Items []User `json:"items"`
		Total int    `json:"total"`
		Page  int    `json:"page"`
		Limit int    `json:"limit"`
	} `json:"body"`
}

type GetUserInput struct {
	ID int `path:"id" validate:"required,min=1"`
}

type GetUserOutput struct {
	Body User `json:"body"`
}

type CreateUserInput struct {
	Body CreateUserBody `json:"body"`
}

type CreateUserOutput struct {
	Body User `json:"body"`
}

type UpdateUserInput struct {
	ID   int            `path:"id"`
	Body UpdateUserBody `json:"body"`
}

type UpdateUserOutput struct {
	Body User `json:"body"`
}

type DeleteUserInput struct {
	ID int `path:"id"`
}

type DeleteUserOutput struct {
	Body struct {
		Deleted bool `json:"deleted"`
	} `json:"body"`
}

type HealthOutput struct {
	Body struct {
		Status string `json:"status"`
		Time   string `json:"time"`
	} `json:"body"`
}

func RegisterUserRoutes(server httpx.ServerRuntime, service UserService) {
	if server == nil || service == nil {
		return
	}

	httpx.MustGet(server, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		out := &HealthOutput{}
		out.Body.Status = "ok"
		out.Body.Time = time.Now().UTC().Format(time.RFC3339)
		return out, nil
	}, huma.OperationTags("system"))

	api := server.Group("/api/v1")

	httpx.MustGroupGet(api, "/users", func(ctx context.Context, input *ListUsersInput) (*ListUsersOutput, error) {
		limit := input.Limit
		if limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}
		page := input.Page
		if page <= 0 {
			page = 1
		}

		offset := (page - 1) * limit
		items, total := service.List(input.Q, limit, offset)
		out := &ListUsersOutput{}
		out.Body.Items = items
		out.Body.Total = total
		out.Body.Page = page
		out.Body.Limit = limit
		return out, nil
	}, huma.OperationTags("users"))

	httpx.MustGroupGet(api, "/users/{id}", func(ctx context.Context, input *GetUserInput) (*GetUserOutput, error) {
		user, ok := service.Get(input.ID)
		if !ok {
			return nil, httpx.NewError(404, "user not found")
		}
		out := &GetUserOutput{}
		out.Body = user
		return out, nil
	}, huma.OperationTags("users"))

	httpx.MustGroupPost(api, "/users", func(ctx context.Context, input *CreateUserInput) (*CreateUserOutput, error) {
		user := service.Create(input.Body)
		out := &CreateUserOutput{}
		out.Body = user
		return out, nil
	}, huma.OperationTags("users"))

	httpx.MustGroupPut(api, "/users/{id}", func(ctx context.Context, input *UpdateUserInput) (*UpdateUserOutput, error) {
		user, ok := service.Update(input.ID, input.Body)
		if !ok {
			return nil, httpx.NewError(404, "user not found")
		}
		out := &UpdateUserOutput{}
		out.Body = user
		return out, nil
	}, huma.OperationTags("users"))

	httpx.MustGroupDelete(api, "/users/{id}", func(ctx context.Context, input *DeleteUserInput) (*DeleteUserOutput, error) {
		deleted := service.Delete(input.ID)
		if !deleted {
			return nil, httpx.NewError(404, "user not found")
		}
		out := &DeleteUserOutput{}
		out.Body.Deleted = true
		return out, nil
	}, huma.OperationTags("users"))
}
