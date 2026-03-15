package eventsub

import (
	"context"
	"fmt"
	"log/slog"

	collectionlist "github.com/DaiYuANg/archgo/collectionx/list"
	"github.com/DaiYuANg/archgo/eventx"
	endpointevents "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/events"
	"github.com/DaiYuANg/archgo/logx"
	"github.com/DaiYuANg/archgo/observabilityx"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"go.uber.org/fx"
)

func Register(
	lc fx.Lifecycle,
	bus eventx.BusRuntime,
	logger *slog.Logger,
	obs observabilityx.Observability,
) error {
	unsubscribers := collectionlist.NewList[func()]()

	loginUnsub, err := subscribeWithMetrics(
		bus,
		obs,
		logger,
		"subscribe login event failed",
		func(event endpointevents.LoginSucceededEvent) map[string]any {
			return map[string]any{
				"user_id":  event.UserID,
				"username": event.Username,
				"roles":    event.Roles,
			}
		},
	)
	if err != nil {
		return err
	}
	unsubscribers.Add(loginUnsub)

	createdUnsub, err := subscribeWithMetrics(
		bus,
		obs,
		logger,
		"subscribe book created event failed",
		func(event endpointevents.BookCreatedEvent) map[string]any {
			return map[string]any{
				"book_id":  event.BookID,
				"title":    event.Title,
				"actor_id": event.ActorID,
				"actor":    event.Actor,
			}
		},
	)
	if err != nil {
		return err
	}
	unsubscribers.Add(createdUnsub)

	deletedUnsub, err := subscribeWithMetrics(
		bus,
		obs,
		logger,
		"subscribe book deleted event failed",
		func(event endpointevents.BookDeletedEvent) map[string]any {
			return map[string]any{
				"book_id":  event.BookID,
				"actor_id": event.ActorID,
				"actor":    event.Actor,
			}
		},
	)
	if err != nil {
		return err
	}
	unsubscribers.Add(deletedUnsub)

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			unsubscribers.Range(func(_ int, unsubscribe func()) bool {
				mo.TupleToOption(unsubscribe, unsubscribe != nil).ForEach(func(fn func()) {
					fn()
				})
				return true
			})
			return nil
		},
	})

	return nil
}

func subscribeWithMetrics[T eventx.Event](
	bus eventx.BusRuntime,
	obs observabilityx.Observability,
	logger *slog.Logger,
	errMessage string,
	fields func(T) map[string]any,
) (func(), error) {
	unsub, err := eventx.Subscribe(bus, func(ctx context.Context, event T) error {
		obs.AddCounter(ctx, "rbac_events_total", 1, observabilityx.String("event", event.Name()))

		extra := lo.Ternary(fields != nil, fields(event), map[string]any{})
		logx.WithFields(logger, lo.Assign(map[string]any{"event": event.Name()}, extra)).Info("event handled")
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMessage, err)
	}
	return unsub, nil
}
