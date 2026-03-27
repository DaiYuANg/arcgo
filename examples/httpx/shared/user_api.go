package shared

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/danielgtaylor/huma/v2"
)

type listUsersInput struct {
	Limit int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Page  int    `query:"page"  validate:"omitempty,min=1"`
	Q     string `query:"q"     validate:"omitempty,max=100"`
}

type listUsersOutput struct {
	Body struct {
		Items []User `json:"items"`
		Total int    `json:"total"`
		Page  int    `json:"page"`
		Limit int    `json:"limit"`
	} `json:"body"`
}

type getUserInput struct {
	ID int `path:"id" validate:"required,min=1"`
}

type getUserOutput struct {
	Body User `json:"body"`
}

type createUserInput struct {
	Body CreateUserBody `json:"body"`
}

type createUserOutput struct {
	Body User `json:"body"`
}

type updateUserInput struct {
	ID   int            `path:"id"`
	Body UpdateUserBody `json:"body"`
}

type updateUserOutput struct {
	Body User `json:"body"`
}

type deleteUserInput struct {
	ID int `path:"id"`
}

type deleteUserOutput struct {
	Body struct {
		Deleted bool `json:"deleted"`
	} `json:"body"`
}

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
		Time   string `json:"time"`
	} `json:"body"`
}

// RegisterUserRoutes registers the shared user demo routes on the server.
func RegisterUserRoutes(server httpx.ServerRuntime, service UserService) {
	if server == nil || service == nil {
		return
	}

	registerHealthRoute(server)
	api := server.Group("/api/v1")
	registerListUsersRoute(api, service)
	registerGetUserRoute(api, service)
	registerCreateUserRoute(api, service)
	registerUpdateUserRoute(api, service)
	registerDeleteUserRoute(api, service)
}

func registerHealthRoute(server httpx.ServerRuntime) {
	httpx.MustGet(server, "/health", func(_ context.Context, _ *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "ok"
		out.Body.Time = time.Now().UTC().Format(time.RFC3339)
		return out, nil
	}, huma.OperationTags("system"))
}

func registerListUsersRoute(api *httpx.Group, service UserService) {
	httpx.MustGroupGet(api, "/users", func(_ context.Context, input *listUsersInput) (*listUsersOutput, error) {
		limit, page := normalizePagination(input.Limit, input.Page)
		offset := (page - 1) * limit
		items, total := service.List(input.Q, limit, offset)
		out := &listUsersOutput{}
		out.Body.Items = items
		out.Body.Total = total
		out.Body.Page = page
		out.Body.Limit = limit
		return out, nil
	}, huma.OperationTags("users"))
}

func registerGetUserRoute(api *httpx.Group, service UserService) {
	httpx.MustGroupGet(api, "/users/{id}", func(_ context.Context, input *getUserInput) (*getUserOutput, error) {
		user, ok := service.Get(input.ID)
		if !ok {
			return nil, httpx.NewError(404, "user not found")
		}
		out := &getUserOutput{}
		out.Body = user
		return out, nil
	}, huma.OperationTags("users"))
}

func registerCreateUserRoute(api *httpx.Group, service UserService) {
	httpx.MustGroupPost(api, "/users", func(_ context.Context, input *createUserInput) (*createUserOutput, error) {
		user := service.Create(input.Body)
		out := &createUserOutput{}
		out.Body = user
		return out, nil
	}, huma.OperationTags("users"))
}

func registerUpdateUserRoute(api *httpx.Group, service UserService) {
	httpx.MustGroupPut(api, "/users/{id}", func(_ context.Context, input *updateUserInput) (*updateUserOutput, error) {
		user, ok := service.Update(input.ID, input.Body)
		if !ok {
			return nil, httpx.NewError(404, "user not found")
		}
		out := &updateUserOutput{}
		out.Body = user
		return out, nil
	}, huma.OperationTags("users"))
}

func registerDeleteUserRoute(api *httpx.Group, service UserService) {
	httpx.MustGroupDelete(api, "/users/{id}", func(_ context.Context, input *deleteUserInput) (*deleteUserOutput, error) {
		deleted := service.Delete(input.ID)
		if !deleted {
			return nil, httpx.NewError(404, "user not found")
		}
		out := &deleteUserOutput{}
		out.Body.Deleted = true
		return out, nil
	}, huma.OperationTags("users"))
}

func normalizePagination(limit, page int) (int, int) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if page <= 0 {
		page = 1
	}
	return limit, page
}
