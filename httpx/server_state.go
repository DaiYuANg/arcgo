package httpx

import "log/slog"

// IsFrozen reports whether runtime configuration is frozen.
func (s *Server) IsFrozen() bool {
	return s != nil && s.frozen.Load()
}

func (s *Server) freezeConfiguration() {
	if s == nil {
		return
	}
	s.openAPIMu.Lock()
	defer s.openAPIMu.Unlock()
	s.frozen.Store(true)
}

func (s *Server) allowConfigMutation(action string) bool {
	if s == nil {
		return false
	}
	if !s.IsFrozen() {
		return true
	}
	if s.logger != nil {
		s.logger.Warn(
			ErrServerFrozen.Error(),
			slog.String("action", action),
		)
	}
	return false
}
