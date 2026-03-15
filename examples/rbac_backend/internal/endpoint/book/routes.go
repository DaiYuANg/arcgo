package book

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/archgo/authx"
	"github.com/DaiYuANg/archgo/eventx"
	endpointeventpublish "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/eventpublish"
	endpointevents "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/events"
	endpointhttperr "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/httperr"
	endpointoperation "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/operation"
	endpointresponse "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/response"
	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/entity"
	modelbook "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/book"
	booksvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/book"
	"github.com/DaiYuANg/archgo/httpx"
	"github.com/DaiYuANg/archgo/observabilityx"
	"github.com/danielgtaylor/huma/v2"
)

type Endpoint struct {
	httpx.BaseEndpoint
	bookSvc *booksvc.Service
	bus     eventx.BusRuntime
	obs     observabilityx.Observability
	logger  *slog.Logger
}

func NewEndpoint(
	bookSvc *booksvc.Service,
	bus eventx.BusRuntime,
	obs observabilityx.Observability,
	logger *slog.Logger,
) *Endpoint {
	return &Endpoint{
		bookSvc: bookSvc,
		bus:     bus,
		obs:     obs,
		logger:  logger,
	}
}

func (e *Endpoint) RegisterRoutes(server httpx.ServerRuntime) {
	bookSvc := e.bookSvc
	bus := e.bus
	obs := e.obs
	logger := e.logger

	httpx.MustGet(server, "/books", func(ctx context.Context, _ *struct{}) (*modelbook.ListOutput, error) {
		ctx, span, done := endpointoperation.Begin(ctx, obs, "rbac.route.list_books")
		defer done()
		defer span.End()

		items, err := bookSvc.List(ctx)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}

		endpointoperation.CountRouteResult(ctx, obs, "list_books", "ok")
		return endpointresponse.OK(modelbook.ListData{
			Items: items,
			Total: len(items),
		}), nil
	}, huma.OperationTags("book"))

	httpx.MustPost(server, "/books", func(ctx context.Context, input *modelbook.CreateInput) (*modelbook.CreateOutput, error) {
		ctx, span, done := endpointoperation.Begin(ctx, obs, "rbac.route.create_book")
		defer done()
		defer span.End()

		principal, ok := authx.PrincipalFromContextAs[entity.Principal](ctx)
		if !ok {
			return nil, endpointhttperr.Unauthorized("principal not found")
		}

		item, err := bookSvc.Create(ctx, input.Body.Title, input.Body.Author, principal.UserID)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}

		endpointeventpublish.Async(ctx, bus, endpointevents.BookCreatedEvent{
			BookID:  item.ID,
			Title:   item.Title,
			Author:  item.Author,
			ActorID: principal.UserID,
			Actor:   principal.Username,
		}, logger)

		endpointoperation.CountRouteResult(ctx, obs, "create_book", "ok")
		return endpointresponse.OK(item), nil
	}, huma.OperationTags("book"))

	httpx.MustDelete(server, "/books/{id}", func(ctx context.Context, input *modelbook.DeleteInput) (*modelbook.DeleteOutput, error) {
		ctx, span, done := endpointoperation.Begin(ctx, obs, "rbac.route.delete_book")
		defer done()
		defer span.End()

		principal, ok := authx.PrincipalFromContextAs[entity.Principal](ctx)
		if !ok {
			return nil, endpointhttperr.Unauthorized("principal not found")
		}

		deleted, err := bookSvc.Delete(ctx, input.ID)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		if !deleted {
			return nil, endpointhttperr.NotFound("book not found")
		}

		endpointeventpublish.Async(ctx, bus, endpointevents.BookDeletedEvent{
			BookID:  input.ID,
			ActorID: principal.UserID,
			Actor:   principal.Username,
		}, logger)

		endpointoperation.CountRouteResult(ctx, obs, "delete_book", "ok")
		return endpointresponse.OK(modelbook.DeleteData{Deleted: true}), nil
	}, huma.OperationTags("book"))
}
