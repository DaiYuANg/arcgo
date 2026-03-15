package httpapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/config"
	"github.com/DaiYuANg/archgo/httpx"
	"github.com/DaiYuANg/archgo/httpx/adapter"
	httpxfiber "github.com/DaiYuANg/archgo/httpx/adapter/fiber"
	"github.com/DaiYuANg/archgo/logx"
	"github.com/DaiYuANg/archgo/observabilityx"
	promobs "github.com/DaiYuANg/archgo/observabilityx/prometheus"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v2"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/fx"
)

func NewFiberAdapter(
	cfg config.AppConfig,
	logger *slog.Logger,
	obs observabilityx.Observability,
	authMW fiber.Handler,
) *httpxfiber.Adapter {
	fiberAdapter := httpxfiber.NewWithOptions(nil, httpxfiber.Options{
		Logger: logger,
		Huma: adapter.HumaOptions{
			Title:       "ArcGo RBAC Backend Scaffold",
			Version:     cfg.Version,
			Description: "httpx(fiber) + authx(jwt) + eventx + observabilityx + bun + fx",
			DocsPath:    cfg.DocsPath(),
			OpenAPIPath: cfg.OpenAPIPath(),
			Transformers: []huma.Transformer{
				ResultEnvelopeTransformer,
			},
		},
	})

	router := fiberAdapter.Router()
	router.Use(fiberrecover.New())
	router.Use(NewRequestObservabilityMiddleware(obs))
	router.Use(NewRequestLogMiddleware(logger))
	router.Use(authMW)
	return fiberAdapter
}

func NewServerOptions(cfg config.AppConfig, logger *slog.Logger, fiberAdapter *httpxfiber.Adapter) []httpx.ServerOption {
	return []httpx.ServerOption{
		httpx.WithAdapter(fiberAdapter),
		httpx.WithBasePath(cfg.BasePath()),
		httpx.WithLogger(logx.WithFields(logger, map[string]any{"component": "httpx"})),
		httpx.WithOpenAPIInfo(
			"ArcGo RBAC Backend Scaffold",
			cfg.Version,
			"A reusable RBAC backend scaffold built with arcgo packages",
		),
	}
}

func RegisterInfraRoutes(server httpx.ServerRuntime, cfg config.AppConfig, prom *promobs.Adapter) {
	server.Adapter().Handle(httpx.MethodGet, "/health", func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
	) error {
		_ = ctx
		_ = r
		_, err := w.Write([]byte("ok"))
		return err
	})

	server.Adapter().Handle(httpx.MethodGet, cfg.MetricsPath(), func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
	) error {
		_ = ctx
		prom.Handler().ServeHTTP(w, r)
		return nil
	})
}

func StartServer(
	lc fx.Lifecycle,
	cfg config.AppConfig,
	logger *slog.Logger,
	server httpx.ServerRuntime,
) {
	var runCancel context.CancelFunc
	listenErrCh := make(chan error, 1)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			runCtx, cancel := context.WithCancel(context.Background())
			runCancel = cancel

			go func() {
				listenErrCh <- server.ListenAndServeContext(runCtx, cfg.Addr())
			}()

			select {
			case err := <-listenErrCh:
				return err
			case <-time.After(200 * time.Millisecond):
			}

			logger.Info("rbac backend started",
				slog.String("address", cfg.Addr()),
				slog.String("health", fmt.Sprintf("http://127.0.0.1%s/health", cfg.Addr())),
				slog.String("docs", fmt.Sprintf("http://127.0.0.1%s%s", cfg.Addr(), cfg.DocsPath())),
				slog.String("openapi", fmt.Sprintf("http://127.0.0.1%s%s", cfg.Addr(), cfg.OpenAPIPath())),
				slog.String("metrics", fmt.Sprintf("http://127.0.0.1%s%s", cfg.Addr(), cfg.MetricsPath())),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if runCancel != nil {
				runCancel()
			}

			select {
			case err := <-listenErrCh:
				return err
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				return nil
			}
		},
	})
}
