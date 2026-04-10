package httpx

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/pkg/option"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-playground/validator/v10"
)

// Server is the central httpx runtime object used to register routes and expose
// Huma/OpenAPI capabilities.
type Server struct {
	adapter            adapter.Host
	basePath           string
	routes             *list.ConcurrentList[RouteInfo]
	routesByMethod     *mapping.ConcurrentMultiMap[string, RouteInfo]
	routeExact         *mapping.ConcurrentMap[string, RouteInfo]
	routeMatchers      *mapping.ConcurrentMap[string, *routeMatcher]
	logger             *slog.Logger
	printRoutes        bool
	validator          *validator.Validate
	panicRecover       bool
	accessLog          bool
	openAPIPatches     *list.ConcurrentList[func(*huma.OpenAPI)]
	humaMiddlewares    *list.ConcurrentList[func(huma.Context, func(huma.Context))]
	operationModifiers *list.ConcurrentList[func(*huma.Operation)]
	openAPIMu          sync.Mutex
	routeSequence      atomic.Uint64
	frozen             atomic.Bool
}

// ServerOption mutates a server during construction.
type ServerOption func(*Server)

// newServer constructs a server, creating a default std adapter when none is provided.
func newServer(opts ...ServerOption) *Server {
	s := &Server{
		logger:             slog.Default(),
		routes:             list.NewConcurrentList[RouteInfo](),
		routesByMethod:     mapping.NewConcurrentMultiMap[string, RouteInfo](),
		routeExact:         mapping.NewConcurrentMap[string, RouteInfo](),
		routeMatchers:      mapping.NewConcurrentMap[string, *routeMatcher](),
		panicRecover:       true,
		openAPIPatches:     list.NewConcurrentList[func(*huma.OpenAPI)](),
		humaMiddlewares:    list.NewConcurrentList[func(huma.Context, func(huma.Context))](),
		operationModifiers: list.NewConcurrentList[func(*huma.Operation)](),
	}
	option.Apply(s, opts...)

	if s.adapter == nil {
		s.adapter = std.New(nil)
	}

	s.applyPendingHumaConfig()
	if s.logger != nil && s.logger.Enabled(context.Background(), slog.LevelDebug) {
		s.logger.Debug("httpx server created",
			slog.String("adapter", s.adapter.Name()),
			slog.String("base_path", s.basePath),
			slog.Bool("panic_recover", s.panicRecover),
			slog.Bool("access_log", s.accessLog),
			slog.Bool("print_routes", s.printRoutes),
		)
	}

	return s
}
