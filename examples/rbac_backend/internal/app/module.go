package app

import (
	authxfx "github.com/DaiYuANg/archgo/authx/fx"
	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/app/eventsub"
	appproviders "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/app/providers"
	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/authn"
	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/config"
	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint"
	httpapp "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/http"
	repoauth "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/repository/auth"
	repobook "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/repository/book"
	repocore "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/repository/core"
	reporole "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/repository/role"
	repouser "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/repository/user"
	authsvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/auth"
	booksvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/book"
	rolesvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/role"
	usersvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/user"
	httpxfx "github.com/DaiYuANg/archgo/httpx/fx"
	"github.com/DaiYuANg/archgo/logx"
	logxfx "github.com/DaiYuANg/archgo/logx/fx"
	"go.uber.org/fx"
)

func newAppModule() fx.Option {
	return fx.Options(
		logxfx.NewLogxModule(
			logx.WithConsole(true),
			logx.WithInfoLevel(),
		),
		authxfx.NewAuthxModule(),
		httpxfx.NewHttpxModule(),
		fx.Provide(
			config.New,
			appproviders.NewPrometheusAdapter,
			appproviders.NewObservability,
			authsvc.NewJWTService,
			appproviders.NewEventBus,
			repocore.NewStore,
			repoauth.NewRepository,
			repoauth.NewAuthorizationRepository,
			repobook.NewRepository,
			repouser.NewRepository,
			reporole.NewRepository,
			authsvc.NewService,
			authsvc.NewAuthorizationService,
			booksvc.NewService,
			usersvc.NewService,
			rolesvc.NewService,
			fx.Annotate(authn.NewAuthxEngineOptions, fx.ResultTags(`group:"authx_engine_options,flatten"`)),
			authn.NewGuard,
			authn.NewAuthMiddleware,
			httpapp.NewFiberAdapter,
			fx.Annotate(httpapp.NewServerOptions, fx.ResultTags(`group:"httpx_server_options,flatten"`)),
		),
		fx.Invoke(
			eventsub.Register,
			endpoint.RegisterHTTPRoutes,
			httpapp.RegisterInfraRoutes,
			httpapp.StartServer,
		),
	)
}
