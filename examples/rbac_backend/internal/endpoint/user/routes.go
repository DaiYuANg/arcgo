package user

import (
	"context"
	"database/sql"
	"errors"

	endpointhttperr "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/httperr"
	endpointoperation "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/operation"
	endpointresponse "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/response"
	modeluser "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/user"
	usersvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/user"
	"github.com/DaiYuANg/archgo/httpx"
	"github.com/DaiYuANg/archgo/observabilityx"
	"github.com/danielgtaylor/huma/v2"
)

type Endpoint struct {
	httpx.BaseEndpoint
	userSvc *usersvc.Service
	obs     observabilityx.Observability
}

func NewEndpoint(userSvc *usersvc.Service, obs observabilityx.Observability) *Endpoint {
	return &Endpoint{
		userSvc: userSvc,
		obs:     obs,
	}
}

func (e *Endpoint) RegisterRoutes(server httpx.ServerRuntime) {
	userSvc := e.userSvc
	obs := e.obs

	httpx.MustGet(server, "/users", func(ctx context.Context, _ *struct{}) (*modeluser.ListOutput, error) {
		ctx, span, done := endpointoperation.Begin(ctx, obs, "rbac.route.list_users")
		defer done()
		defer span.End()

		items, err := userSvc.List(ctx)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		return endpointresponse.OK(modeluser.ListData{
			Items: items,
			Total: len(items),
		}), nil
	}, huma.OperationTags("user"))

	httpx.MustGet(server, "/users/{id}", func(ctx context.Context, input *modeluser.GetInput) (*modeluser.GetOutput, error) {
		item, err := userSvc.Get(ctx, input.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, endpointhttperr.NotFound("user not found")
			}
			return nil, err
		}
		return endpointresponse.OK(item), nil
	}, huma.OperationTags("user"))

	httpx.MustPost(server, "/users", func(ctx context.Context, input *modeluser.CreateInput) (*modeluser.CreateOutput, error) {
		item, err := userSvc.Create(ctx, usersvc.CreateCommand{
			Username:  input.Body.Username,
			Password:  input.Body.Password,
			RoleCodes: input.Body.RoleCodes,
		})
		if err != nil {
			return nil, err
		}
		return endpointresponse.OK(item), nil
	}, huma.OperationTags("user"))

	httpx.MustPut(server, "/users/{id}", func(ctx context.Context, input *modeluser.UpdateInput) (*modeluser.UpdateOutput, error) {
		item, err := userSvc.Update(ctx, input.ID, usersvc.UpdateCommand{
			Username:  input.Body.Username,
			Password:  input.Body.Password,
			RoleCodes: input.Body.RoleCodes,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, endpointhttperr.NotFound("user not found")
			}
			return nil, err
		}
		return endpointresponse.OK(item), nil
	}, huma.OperationTags("user"))

	httpx.MustDelete(server, "/users/{id}", func(ctx context.Context, input *modeluser.DeleteInput) (*modeluser.DeleteOutput, error) {
		deleted, err := userSvc.Delete(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		if !deleted {
			return nil, endpointhttperr.NotFound("user not found")
		}
		return endpointresponse.OK(modeluser.DeleteData{Deleted: true}), nil
	}, huma.OperationTags("user"))
}
