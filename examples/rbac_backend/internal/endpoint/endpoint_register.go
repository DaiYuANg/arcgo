package endpoint

import (
	"log/slog"

	"github.com/DaiYuANg/archgo/eventx"
	endpointauth "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/auth"
	endpointbook "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/book"
	endpointrole "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/role"
	endpointuser "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/endpoint/user"
	authsvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/auth"
	booksvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/book"
	rolesvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/role"
	usersvc "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/service/user"
	"github.com/DaiYuANg/archgo/httpx"
	"github.com/DaiYuANg/archgo/observabilityx"
)

func RegisterHTTPRoutes(
	server httpx.ServerRuntime,
	authSvc *authsvc.Service,
	bookSvc *booksvc.Service,
	userSvc *usersvc.Service,
	roleSvc *rolesvc.Service,
	bus eventx.BusRuntime,
	obs observabilityx.Observability,
	logger *slog.Logger,
) {
	server.RegisterOnly(
		endpointauth.NewEndpoint(authSvc, bus, obs, logger),
		endpointbook.NewEndpoint(bookSvc, bus, obs, logger),
		endpointuser.NewEndpoint(userSvc, obs),
		endpointrole.NewEndpoint(roleSvc, obs),
	)
}
