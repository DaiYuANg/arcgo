package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
)

// Handler returns the server as an `http.Handler`.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.accessLog {
			s.adapter.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		recorder := newAccessLogResponseWriter(w)
		s.adapter.ServeHTTP(recorder, r)
		s.logRequest(r, recorder.Status(), time.Since(start))
	})
}

// ServeHTTP delegates request handling to the underlying adapter.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Handler().ServeHTTP(w, r)
}

// ListenAndServe starts related services.
func (s *Server) ListenAndServe(addr string) error {
	routeCount := s.RouteCount()
	s.logger.Info("Starting server",
		slog.String("address", addr),
		slog.String("adapter", s.adapter.Name()),
		slog.Int("routes", routeCount),
	)

	if listenable, ok := s.adapter.(adapter.ListenableAdapter); ok {
		if err := listenable.Listen(addr); err != nil {
			return fmt.Errorf("httpx: adapter %q listen on %q: %w", s.adapter.Name(), addr, err)
		}
		return nil
	}

	if err := http.ListenAndServe(addr, s.Handler()); err != nil {
		return fmt.Errorf("httpx: listen on %q: %w", addr, err)
	}
	return nil
}

// ListenAndServeContext starts related services.
func (s *Server) ListenAndServeContext(ctx context.Context, addr string) error {
	if listenable, ok := s.adapter.(adapter.ContextListenableAdapter); ok {
		return listenable.ListenContext(ctx, addr)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}

	s.logger.Info("Starting server with context", slog.String("address", addr))

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpx: listen on %q: %w", addr, err)
	case <-ctx.Done():
		s.logger.Info("Shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("httpx: shutdown server on %q: %w", addr, err)
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpx: listen on %q: %w", addr, err)
	}
}

func (s *Server) logRequest(r *http.Request, status int, duration time.Duration) {
	if s == nil || s.logger == nil || r == nil {
		return
	}

	attrs := []any{
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
		slog.Duration("duration", duration),
	}

	if route, ok := s.matchRoute(r.Method, r.URL.Path); ok {
		attrs = append(attrs,
			slog.String("route", route.Path),
			slog.String("handler", route.HandlerName),
		)
	}

	s.logger.Info("httpx request", attrs...)
}
