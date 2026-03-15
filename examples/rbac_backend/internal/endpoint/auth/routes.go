package auth

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/archgo/eventx"
	endpointeventpublish "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/eventpublish"
	endpointevents "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/events"
	endpointhttperr "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/httperr"
	endpointoperation "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/operation"
	endpointresponse "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/response"
	modelauth "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/auth"
	authsvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/auth"
	"github.com/DaiYuANg/archgo/httpx"
	"github.com/DaiYuANg/archgo/observabilityx"
	"github.com/danielgtaylor/huma/v2"
)

type Endpoint struct {
	httpx.BaseEndpoint
	authSvc *authsvc.Service
	bus     eventx.BusRuntime
	obs     observabilityx.Observability
	logger  *slog.Logger
}

func NewEndpoint(
	authSvc *authsvc.Service,
	bus eventx.BusRuntime,
	obs observabilityx.Observability,
	logger *slog.Logger,
) *Endpoint {
	return &Endpoint{
		authSvc: authSvc,
		bus:     bus,
		obs:     obs,
		logger:  logger,
	}
}

func (e *Endpoint) RegisterRoutes(server httpx.ServerRuntime) {
	authSvc := e.authSvc
	bus := e.bus
	obs := e.obs
	logger := e.logger

	httpx.MustPost(server, "/login", func(ctx context.Context, input *modelauth.LoginInput) (*modelauth.LoginOutput, error) {
		ctx, span, done := endpointoperation.Begin(ctx, obs, "rbac.route.login")
		defer done()
		defer span.End()

		principal, token, err := authSvc.Login(ctx, input.Body.Username, input.Body.Password)
		if err != nil {
			span.RecordError(err)
			endpointoperation.CountRouteResult(ctx, obs, "login", "denied")
			return nil, endpointhttperr.Unauthorized("invalid username or password")
		}

		endpointeventpublish.Async(ctx, bus, endpointevents.LoginSucceededEvent{
			UserID:   principal.UserID,
			Username: principal.Username,
			Roles:    principal.Roles,
		}, logger)

		endpointoperation.CountRouteResult(ctx, obs, "login", "ok")
		return endpointresponse.OK(modelauth.LoginData{
			Token:    token,
			UserID:   principal.UserID,
			Username: principal.Username,
			Roles:    principal.Roles,
		}), nil
	}, huma.OperationTags("auth"))
}
