//go:build !no_fiber

package fiber

import (
	"context"
	"net/http"
	"net/url"

	"github.com/DaiYuANg/arcgo/authx"
	authhttp "github.com/DaiYuANg/arcgo/authx/http"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

// Option configures fiber middleware behavior.
type Option func(*config)

type config struct {
	failureHandler func(*fiber.Ctx, int, string) error
}

func defaultConfig() config {
	return config{
		failureHandler: func(c *fiber.Ctx, status int, message string) error {
			return c.Status(status).JSON(fiber.Map{"error": message})
		},
	}
}

// WithFailureHandler overrides the default auth failure handler.
func WithFailureHandler(handler func(*fiber.Ctx, int, string) error) Option {
	return func(cfg *config) {
		if handler != nil {
			cfg.failureHandler = handler
		}
	}
}

// Require runs Check + Can and writes failure response automatically when denied.
func Require(guard *authhttp.Guard, opts ...Option) fiber.Handler {
	return requireWithMode(guard, false, opts...)
}

// RequireFast runs Check + Can with fast request extraction (less copying).
func RequireFast(guard *authhttp.Guard, opts ...Option) fiber.Handler {
	return requireWithMode(guard, true, opts...)
}

func requireWithMode(guard *authhttp.Guard, fast bool, opts ...Option) fiber.Handler {
	cfg := defaultConfig()
	authhttp.ApplyOptions(&cfg, opts...)
	extract := requestInfoFromFiber
	if fast {
		extract = requestInfoFromFiberFast
	}

	return func(c *fiber.Ctx) error {
		if guard == nil {
			return cfg.failureHandler(c, http.StatusInternalServerError, "internal_error")
		}

		ctx := c.UserContext()
		if ctx == nil {
			ctx = context.Background()
		}

		reqInfo := extract(c)
		result, decision, err := guard.Require(ctx, reqInfo)
		if err != nil {
			return cfg.failureHandler(c, authhttp.StatusCodeFromError(err), authhttp.ErrorMessage(err))
		}
		if !decision.Allowed {
			return cfg.failureHandler(c, http.StatusForbidden, authhttp.DeniedMessage(decision))
		}

		c.SetUserContext(authx.WithPrincipal(ctx, result.Principal))
		return c.Next()
	}
}

func requestInfoFromFiberFast(c *fiber.Ctx) authhttp.RequestInfo {
	pattern := c.Path()
	if route := c.Route(); route != nil {
		if route.Path != "" {
			pattern = route.Path
		}
	}

	return authhttp.RequestInfo{
		Method:       c.Method(),
		Path:         c.Path(),
		RoutePattern: pattern,
		Headers:      nil,
		Query:        nil,
		PathParams:   nil,
		Request:      nil,
		Native:       c,
	}
}

func requestInfoFromFiber(c *fiber.Ctx) authhttp.RequestInfo {
	pattern := c.Path()
	if route := c.Route(); route != nil {
		if route.Path != "" {
			pattern = route.Path
		}
	}

	headers := make(http.Header)
	for key, value := range c.Request().Header.All() {
		headers.Add(string(key), string(value))
	}

	var query url.Values
	if len(c.Request().URI().QueryString()) > 0 {
		query = make(url.Values)
		for key, value := range c.Request().URI().QueryArgs().All() {
			query.Add(string(key), string(value))
		}
	}

	var pathParams map[string]string
	if route := c.Route(); route != nil && len(route.Params) > 0 {
		pathParams = lo.Associate(route.Params, func(key string) (string, string) {
			return key, c.Params(key)
		})
	}

	return authhttp.RequestInfo{
		Method:       c.Method(),
		Path:         c.Path(),
		RoutePattern: pattern,
		Headers:      headers,
		Query:        query,
		PathParams:   pathParams,
		Request:      nil,
		Native:       c,
	}
}
