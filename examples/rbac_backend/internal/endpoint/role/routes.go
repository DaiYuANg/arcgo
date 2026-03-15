package role

import (
	"context"
	"database/sql"
	"errors"

	endpointhttperr "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/httperr"
	endpointoperation "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/operation"
	endpointresponse "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/response"
	modelrole "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/role"
	rolesvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/role"
	"github.com/DaiYuANg/archgo/httpx"
	"github.com/DaiYuANg/archgo/observabilityx"
	"github.com/danielgtaylor/huma/v2"
)

type Endpoint struct {
	httpx.BaseEndpoint
	roleSvc *rolesvc.Service
	obs     observabilityx.Observability
}

func NewEndpoint(roleSvc *rolesvc.Service, obs observabilityx.Observability) *Endpoint {
	return &Endpoint{
		roleSvc: roleSvc,
		obs:     obs,
	}
}

func (e *Endpoint) RegisterRoutes(server httpx.ServerRuntime) {
	roleSvc := e.roleSvc
	obs := e.obs

	httpx.MustGet(server, "/roles", func(ctx context.Context, _ *struct{}) (*modelrole.ListOutput, error) {
		ctx, span, done := endpointoperation.Begin(ctx, obs, "rbac.route.list_roles")
		defer done()
		defer span.End()

		items, err := roleSvc.List(ctx)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		return endpointresponse.OK(modelrole.ListData{
			Items: items,
			Total: len(items),
		}), nil
	}, huma.OperationTags("role"))

	httpx.MustGet(server, "/roles/{id}", func(ctx context.Context, input *modelrole.GetInput) (*modelrole.GetOutput, error) {
		item, err := roleSvc.Get(ctx, input.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, endpointhttperr.NotFound("role not found")
			}
			return nil, err
		}
		return endpointresponse.OK(item), nil
	}, huma.OperationTags("role"))

	httpx.MustPost(server, "/roles", func(ctx context.Context, input *modelrole.CreateInput) (*modelrole.CreateOutput, error) {
		item, err := roleSvc.Create(ctx, rolesvc.CreateCommand{Code: input.Body.Code, Name: input.Body.Name})
		if err != nil {
			return nil, err
		}
		return endpointresponse.OK(item), nil
	}, huma.OperationTags("role"))

	httpx.MustPut(server, "/roles/{id}", func(ctx context.Context, input *modelrole.UpdateInput) (*modelrole.UpdateOutput, error) {
		item, err := roleSvc.Update(ctx, input.ID, rolesvc.UpdateCommand{Code: input.Body.Code, Name: input.Body.Name})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, endpointhttperr.NotFound("role not found")
			}
			return nil, err
		}
		return endpointresponse.OK(item), nil
	}, huma.OperationTags("role"))

	httpx.MustDelete(server, "/roles/{id}", func(ctx context.Context, input *modelrole.DeleteInput) (*modelrole.DeleteOutput, error) {
		deleted, err := roleSvc.Delete(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		if !deleted {
			return nil, endpointhttperr.NotFound("role not found")
		}
		return endpointresponse.OK(modelrole.DeleteData{Deleted: true}), nil
	}, huma.OperationTags("role"))
}
