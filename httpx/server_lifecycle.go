//revive:disable:file-length-limit Server lifecycle helpers are kept together to preserve related runtime behavior.

package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/samber/oops"
)

// ListenPort starts related services on the provided port.
func (s *Server) ListenPort(port int) error {
	if s == nil {
		return oops.In("httpx").
			With("op", "listen_port", "port", port).
			New("server is nil")
	}
	if port < 0 || port > 65535 {
		return oops.In("httpx").
			With("op", "listen_port", "port", port).
			Errorf("httpx: invalid port %d", port)
	}
	return s.Listen(fmt.Sprintf(":%d", port))
}

// Shutdown stops related services through the underlying adapter.
func (s *Server) Shutdown() error {
	if s == nil || s.adapter == nil {
		return oops.In("httpx").
			With("op", "shutdown").
			Wrapf(ErrAdapterNotFound, "adapter does not support shutdown")
	}
	if s.logger != nil && s.logger.Enabled(context.Background(), slog.LevelDebug) {
		s.logger.Debug("httpx shutdown requested",
			slog.String("adapter", s.adapter.Name()),
		)
	}

	var shutdownErr error
	if useHostCapability(s, func(shutdownable adapter.ShutdownAdapter) {
		shutdownErr = shutdownable.Shutdown()
	}) {
		if shutdownErr != nil && s.logger != nil {
			s.logger.Error("httpx shutdown failed",
				slog.String("adapter", s.adapter.Name()),
				slog.String("error", shutdownErr.Error()),
			)
			return oops.In("httpx").
				With("op", "shutdown", "adapter", s.adapter.Name()).
				Wrapf(shutdownErr, "shutdown adapter")
		} else if s.logger != nil && s.logger.Enabled(context.Background(), slog.LevelDebug) {
			s.logger.Debug("httpx shutdown completed",
				slog.String("adapter", s.adapter.Name()),
			)
		}
		return nil
	}

	return oops.In("httpx").
		With("op", "shutdown", "adapter", s.adapter.Name()).
		Wrapf(ErrAdapterNotFound, "adapter does not support shutdown")
}

// Listen starts related services.
func (s *Server) Listen(addr string) error {
	if s == nil {
		return oops.In("httpx").
			With("op", "listen", "addr", addr).
			New("server is nil")
	}
	s.freezeConfiguration(context.TODO())

	name := "unknown"
	if s.adapter != nil {
		name = s.adapter.Name()
	}

	s.logger.Info("Starting server",
		slog.String("address", addr),
		slog.String("adapter", name),
		slog.Int("routes", s.RouteCount()),
	)
	logServerDebug(s.logger, "httpx listen entering adapter",
		slog.String("address", addr),
		slog.String("adapter", name),
	)

	var listenErr error
	if useHostCapability(s, func(listenable adapter.ListenableAdapter) {
		listenErr = listenable.Listen(addr)
	}) {
		if listenErr != nil {
			s.logger.Error("httpx listen failed",
				slog.String("address", addr),
				slog.String("adapter", name),
				slog.String("error", listenErr.Error()),
			)
			return oops.In("httpx").
				With("op", "listen", "addr", addr, "adapter", name).
				Wrapf(listenErr, "adapter listen")
		}
		logServerDebug(s.logger, "httpx listen returned",
			slog.String("address", addr),
			slog.String("adapter", name),
		)
		return nil
	}

	return oops.In("httpx").
		With("op", "listen", "addr", addr, "adapter", name).
		Wrapf(ErrAdapterNotFound, "adapter does not support direct listening")
}

// ListenAndServe starts related services.
func (s *Server) ListenAndServe(addr string) error {
	return s.Listen(addr)
}

// ListenAndServeContext starts related services.
func (s *Server) ListenAndServeContext(ctx context.Context, addr string) error {
	if s == nil {
		return oops.In("httpx").
			With("op", "listen_context", "addr", addr).
			New("server is nil")
	}
	s.freezeConfiguration(ctx)
	ctx = normalizeServerContext(ctx)

	name := "unknown"
	if s.adapter != nil {
		name = s.adapter.Name()
	}

	if ok, err := s.listenWithContextCapability(ctx, addr, name); ok {
		return err
	}

	listenable, shutdownable, err := s.contextFallbackAdapters(name)
	if err != nil {
		return err
	}
	return s.listenWithContextFallback(ctx, addr, name, listenable, shutdownable)
}

func normalizeServerContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func (s *Server) listenWithContextCapability(ctx context.Context, addr, name string) (bool, error) {
	var listenErr error
	if !useHostCapability(s, func(listenable adapter.ContextListenableAdapter) {
		listenErr = listenable.ListenContext(ctx, addr)
	}) {
		return false, nil
	}
	if listenErr != nil {
		return true, oops.In("httpx").
			With("op", "listen_context", "addr", addr, "adapter", name).
			Wrapf(listenErr, "adapter listen with context")
	}
	return true, nil
}

func (s *Server) contextFallbackAdapters(
	name string,
) (adapter.ListenableAdapter, adapter.ShutdownAdapter, error) {
	var listenable adapter.ListenableAdapter
	var shutdownable adapter.ShutdownAdapter
	if !useHostCapability(s, func(host adapter.ListenableAdapter) {
		listenable = host
	}) || !useHostCapability(s, func(host adapter.ShutdownAdapter) {
		shutdownable = host
	}) {
		return nil, nil, oops.In("httpx").
			With("op", "listen_context", "adapter", name).
			Wrapf(ErrAdapterNotFound, "adapter does not support context listening")
	}
	return listenable, shutdownable, nil
}

func (s *Server) listenWithContextFallback(
	ctx context.Context,
	addr string,
	name string,
	listenable adapter.ListenableAdapter,
	shutdownable adapter.ShutdownAdapter,
) error {
	s.logger.Info("Starting server with context",
		slog.String("address", addr),
		slog.String("adapter", name),
	)
	logServerDebugContext(ctx, s.logger, "httpx context listen using fallback adapter path",
		slog.String("address", addr),
		slog.String("adapter", name),
	)

	errCh := listenAsync(listenable, addr)

	select {
	case err := <-errCh:
		return handleFallbackListenResult(ctx, s.logger, addr, name, err)
	case <-ctx.Done():
		return handleFallbackCancellation(ctx, s.logger, addr, name, shutdownable, errCh)
	}
}

func listenAsync(listenable adapter.ListenableAdapter, addr string) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- listenable.Listen(addr)
	}()
	return errCh
}

func handleFallbackListenResult(ctx context.Context, logger *slog.Logger, addr, name string, err error) error {
	if err != nil {
		logServerError(logger, "httpx context listen exited with error",
			slog.String("address", addr),
			slog.String("adapter", name),
			slog.String("error", err.Error()),
		)
		return oops.In("httpx").
			With("op", "listen_context", "addr", addr, "adapter", name).
			Wrapf(err, "adapter listen")
	}

	logServerDebugContext(ctx, logger, "httpx context listen exited",
		slog.String("address", addr),
		slog.String("adapter", name),
	)
	return nil
}

func handleFallbackCancellation(
	ctx context.Context,
	logger *slog.Logger,
	addr string,
	name string,
	shutdownable adapter.ShutdownAdapter,
	errCh <-chan error,
) error {
	logServerDebugContext(ctx, logger, "httpx context canceled; shutting down adapter",
		slog.String("address", addr),
		slog.String("adapter", name),
	)
	if err := shutdownFallbackAdapter(logger, addr, name, shutdownable); err != nil {
		return err
	}

	select {
	case err := <-errCh:
		return handlePostCancelListenResult(ctx, logger, addr, name, err)
	default:
		logServerDebugContext(ctx, logger, "httpx context fallback returning context error",
			slog.String("address", addr),
			slog.String("adapter", name),
		)
		return oops.In("httpx").
			With("op", "listen_context", "addr", addr, "adapter", name).
			Wrapf(ctx.Err(), "context canceled while serving")
	}
}

func shutdownFallbackAdapter(
	logger *slog.Logger,
	addr string,
	name string,
	shutdownable adapter.ShutdownAdapter,
) error {
	if err := shutdownable.Shutdown(); err != nil && !errors.Is(err, ErrAdapterNotFound) {
		logServerError(logger, "httpx context shutdown failed",
			slog.String("address", addr),
			slog.String("adapter", name),
			slog.String("error", err.Error()),
		)
		return oops.In("httpx").
			With("op", "shutdown", "addr", addr, "adapter", name).
			Wrapf(err, "shutdown server")
	}
	return nil
}

func handlePostCancelListenResult(ctx context.Context, logger *slog.Logger, addr, name string, err error) error {
	if err != nil {
		logServerError(logger, "httpx listen returned after context cancellation",
			slog.String("address", addr),
			slog.String("adapter", name),
			slog.String("error", err.Error()),
		)
		return oops.In("httpx").
			With("op", "listen_context", "addr", addr, "adapter", name, "stage", "after_cancel").
			Wrapf(err, "adapter listen after context cancellation")
	}

	logServerDebugContext(ctx, logger, "httpx listen returned after context cancellation",
		slog.String("address", addr),
		slog.String("adapter", name),
	)
	return nil
}

func logServerDebug(logger *slog.Logger, msg string, attrs ...any) {
	if logger != nil {
		logger.Debug(msg, attrs...)
	}
}

func logServerDebugContext(ctx context.Context, logger *slog.Logger, msg string, attrs ...any) {
	if logger != nil {
		logger.DebugContext(normalizeServerContext(ctx), msg, attrs...)
	}
}

func logServerError(logger *slog.Logger, msg string, attrs ...any) {
	if logger != nil {
		logger.Error(msg, attrs...)
	}
}
