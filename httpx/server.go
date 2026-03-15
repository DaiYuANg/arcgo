package httpx

import (
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/DaiYuANg/archgo/collectionx/list"
	"github.com/DaiYuANg/archgo/collectionx/mapping"
	"github.com/DaiYuANg/archgo/collectionx/set"
	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/DaiYuANg/archgo/httpx/adapter/std"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

// Server is the central httpx runtime object used to register routes and expose
// Huma/OpenAPI capabilities.
type Server struct {
	adapter            adapter.Adapter
	basePath           string
	routes             *list.ConcurrentList[RouteInfo]
	routeKeys          *set.ConcurrentSet[string]
	routesByMethod     *mapping.ConcurrentMultiMap[string, RouteInfo]
	routeExact         *mapping.ConcurrentMap[string, RouteInfo]
	routePatterns      *mapping.ConcurrentMultiMap[string, RouteInfo]
	logger             *slog.Logger
	printRoutes        bool
	validator          *validator.Validate
	panicRecover       bool
	accessLog          bool
	humaOptions        adapter.HumaOptions
	openAPIPatches     *list.ConcurrentList[func(*huma.OpenAPI)]
	humaMiddlewares    *list.ConcurrentList[func(huma.Context, func(huma.Context))]
	operationModifiers *list.ConcurrentList[func(*huma.Operation)]
	openAPIMu          sync.Mutex
	frozen             atomic.Bool
}

// ServerOption mutates a server during construction.
type ServerOption func(*Server)

// newServer constructs a server, creating a default std adapter when none is provided.
func newServer(opts ...ServerOption) *Server {
	s := &Server{
		logger:             slog.Default(),
		routes:             list.NewConcurrentList[RouteInfo](),
		routeKeys:          set.NewConcurrentSet[string](),
		routesByMethod:     mapping.NewConcurrentMultiMap[string, RouteInfo](),
		routeExact:         mapping.NewConcurrentMap[string, RouteInfo](),
		routePatterns:      mapping.NewConcurrentMultiMap[string, RouteInfo](),
		panicRecover:       true,
		openAPIPatches:     list.NewConcurrentList[func(*huma.OpenAPI)](),
		humaMiddlewares:    list.NewConcurrentList[func(huma.Context, func(huma.Context))](),
		operationModifiers: list.NewConcurrentList[func(*huma.Operation)](),
	}

	lo.ForEach(opts, func(opt ServerOption, _ int) {
		opt(s)
	})

	if s.adapter == nil {
		s.adapter = std.New(s.humaOptions)
	}

	s.applyPendingHumaConfig()

	return s
}
