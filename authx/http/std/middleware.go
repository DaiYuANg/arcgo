package std

import (
	"net/http"

	"github.com/DaiYuANg/archgo/authx"
	authhttp "github.com/DaiYuANg/archgo/authx/http"
)

type Option func(*config)

type config struct {
	failureHandler func(http.ResponseWriter, *http.Request, int, string)
}

func defaultConfig() config {
	return config{
		failureHandler: func(w http.ResponseWriter, _ *http.Request, status int, message string) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
		},
	}
}

func WithFailureHandler(handler func(http.ResponseWriter, *http.Request, int, string)) Option {
	return func(cfg *config) {
		if handler != nil {
			cfg.failureHandler = handler
		}
	}
}

// Require runs Check + Can and writes failure response automatically.
func Require(guard *authhttp.Guard, opts ...Option) func(http.Handler) http.Handler {
	return requireWithMode(guard, false, opts...)
}

// RequireFast runs Check + Can with fast request extraction (less copying).
func RequireFast(guard *authhttp.Guard, opts ...Option) func(http.Handler) http.Handler {
	return requireWithMode(guard, true, opts...)
}

func requireWithMode(guard *authhttp.Guard, fast bool, opts ...Option) func(http.Handler) http.Handler {
	cfg := defaultConfig()
	authhttp.ApplyOptions(&cfg, opts...)
	extract := authhttp.RequestInfoFromHTTPRequest
	if fast {
		extract = authhttp.RequestInfoFromHTTPRequestFast
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if guard == nil {
				cfg.failureHandler(w, r, http.StatusInternalServerError, "internal_error")
				return
			}

			result, decision, err := guard.Require(r.Context(), extract(r))
			if err != nil {
				cfg.failureHandler(w, r, authhttp.StatusCodeFromError(err), authhttp.ErrorMessage(err))
				return
			}
			if !decision.Allowed {
				cfg.failureHandler(w, r, http.StatusForbidden, authhttp.DeniedMessage(decision))
				return
			}

			next.ServeHTTP(w, r.WithContext(authx.WithPrincipal(r.Context(), result.Principal)))
		})
	}
}
