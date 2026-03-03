//go:build !no_fiber

package fiber

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/DaiYuANg/toolkit4go/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

// Adapter Fiber v2 框架适配器
//
// 使用方式：
// 1. 创建适配器：fiberAdapter := fiber.New()
// 2. 注册 Fiber 原生中间件：fiberAdapter.App().Use(fiber.Logger(), yourMiddleware...)
// 3. 创建 httpx server 并注册路由
type Adapter struct {
	app     *fiber.App
	group   fiber.Router
	logger  *slog.Logger
	huma    huma.API
	humaCfg adapter.HumaOptions
}

// New 创建 Fiber 适配器
func New(app ...*fiber.App) *Adapter {
	var a *fiber.App
	if len(app) > 0 {
		a = app[0]
	} else {
		a = fiber.New()
	}

	return &Adapter{
		app:    a,
		group:  a,
		logger: slog.Default(),
	}
}

// WithHuma 启用 Huma OpenAPI 文档
func (a *Adapter) WithHuma(opts adapter.HumaOptions) *Adapter {
	a.humaCfg = opts
	a.huma = humafiber.New(a.app, huma.DefaultConfig(opts.Title, opts.Version))

	// Fiber 需要直接注册路由到 app
	a.registerHumaDocs()

	return a
}

// registerHumaDocs 注册 Huma 文档路由到 Fiber
func (a *Adapter) registerHumaDocs() {
	if a.huma == nil {
		return
	}

	openAPIPath := normalizeHumaPath(a.humaCfg.OpenAPIPath, "/openapi.json")
	docsPath := normalizeHumaPath(a.humaCfg.DocsPath, "/docs")

	// OpenAPI JSON
	a.app.Get(openAPIPath, func(c *fiber.Ctx) error {
		c.Type("json")
		return c.JSON(a.huma.OpenAPI())
	})

	// Swagger UI
	a.app.Get(docsPath, func(c *fiber.Ctx) error {
		c.Type("html")
		return c.SendString(a.swaggerUIHTML(openAPIPath))
	})
}

// swaggerUIHTML 生成 Swagger UI HTML
func (a *Adapter) swaggerUIHTML(openAPIPath string) string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>` + a.humaCfg.Title + `</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({url: "` + openAPIPath + `", dom_id: '#swagger-ui'});
    </script>
</body>
</html>`
}

// WithLogger 设置日志记录器
func (a *Adapter) WithLogger(logger *slog.Logger) *Adapter {
	a.logger = logger
	return a
}

// Name 返回适配器名称
func (a *Adapter) Name() string {
	return "fiber"
}

// Handle 注册业务处理函数
func (a *Adapter) Handle(method, path string, handler adapter.HandlerFunc) {
	a.group.Add(method, path, a.wrapHandler(handler))
}

// Group 创建路由组
func (a *Adapter) Group(prefix string) adapter.Adapter {
	return &Adapter{
		app:     a.app,
		group:   a.group.Group(prefix),
		logger:  a.logger,
		huma:    a.huma,
		humaCfg: a.humaCfg,
	}
}

// ServeHTTP 实现 http.Handler 接口（Fiber 不支持）
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "fiber adapter does not support net/http ServeHTTP; use ListenAndServe", http.StatusNotImplemented)
}

// App 返回 Fiber 应用
// 通过此方法可以直接使用 Fiber 的中间件生态
// 例如：adapter.App().Use(fiber.Logger(), yourMiddleware...)
func (a *Adapter) App() *fiber.App {
	return a.app
}

// wrapHandler 包装处理函数为 Fiber 格式
func (a *Adapter) wrapHandler(handler adapter.HandlerFunc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		w := &responseWriter{ctx: c}
		r := convertRequest(c)

		if err := handler(r.Context(), w, r); err != nil {
			a.logger.Error("Handler error",
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("error", err.Error()),
			)
			return err
		}
		return nil
	}
}

// convertRequest 转换 Fiber 请求为标准 http.Request
func convertRequest(c *fiber.Ctx) *http.Request {
	u := &url.URL{
		Path:     c.Path(),
		RawQuery: string(c.Request().URI().QueryString()),
	}

	header := make(http.Header)
	c.Request().Header.VisitAll(func(k, v []byte) {
		header.Add(string(k), string(v))
	})

	req := &http.Request{
		Method:        c.Method(),
		URL:           u,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(c.Body())),
		ContentLength: int64(len(c.Body())),
		Host:          string(c.Request().Header.Host()),
		RemoteAddr:    c.IP(),
	}

	return req.WithContext(userContext(c))
}

// responseWriter 适配 Fiber 响应
type responseWriter struct {
	ctx        *fiber.Ctx
	statusCode int
	header     http.Header
	applied    bool
}

func (w *responseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.ctx.Status(http.StatusOK)
	}
	w.applyHeaders()
	return w.ctx.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ctx.Status(statusCode)
	w.applyHeaders()
}

// HumaAPI 返回 Huma API
func (a *Adapter) HumaAPI() huma.API {
	return a.huma
}

// HasHuma 检查是否启用了 Huma
func (a *Adapter) HasHuma() bool {
	return a.huma != nil
}

func (w *responseWriter) applyHeaders() {
	if w.applied || w.header == nil {
		return
	}
	lo.ForEach(lo.Keys(w.header), func(key string, _ int) {
		values := w.header[key]
		w.ctx.Response().Header.Del(key)
		lo.ForEach(values, func(value string, _ int) {
			w.ctx.Response().Header.Add(key, value)
		})
	})
	w.applied = true
}

func userContext(c *fiber.Ctx) context.Context {
	ctx := c.UserContext()
	return mo.TupleToOption(ctx, ctx != nil).OrElse(context.Background())
}

func normalizeHumaPath(path, fallback string) string {
	trimmed := strings.TrimSpace(path)
	p := mo.TupleToOption(trimmed, trimmed != "").OrElse(fallback)
	return lo.Ternary(strings.HasPrefix(p, "/"), p, "/"+p)
}
