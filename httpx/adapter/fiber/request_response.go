package fiber

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

// wrapHandler adapts an httpx handler to a fiber handler.
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
			return fmt.Errorf("httpx/fiber: handler failed: %w", err)
		}
		return nil
	}
}

// convertRequest converts a fiber request into an `*http.Request`.
func convertRequest(c *fiber.Ctx) *http.Request {
	u := &url.URL{
		Path:     c.Path(),
		RawQuery: string(c.Request().URI().QueryString()),
	}

	header := make(http.Header)
	for k, v := range c.Request().Header.All() {
		header.Add(string(k), string(v))
	}

	req := &http.Request{
		Method:        c.Method(),
		URL:           u,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(c.Body())),
		ContentLength: int64(len(c.Body())),
		Host:          string(c.Request().Header.Host()),
		RemoteAddr:    c.IP(),
	}

	return req.WithContext(adapter.WithRouteParams(userContext(c), c.AllParams()))
}

// responseWriter adapts a fiber response to the `http.ResponseWriter` shape.
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
