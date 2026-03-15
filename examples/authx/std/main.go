package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/DaiYuANg/archgo/authx"
	"github.com/DaiYuANg/archgo/authx/http/examples/shared"
	authstd "github.com/DaiYuANg/archgo/authx/http/std"
	"github.com/DaiYuANg/archgo/logx"
	"github.com/go-chi/chi/v5"
)

func main() {
	logger := logx.MustNew(logx.WithConsole(true), logx.WithInfoLevel()).With("example", "authx-http-std")
	guard := shared.NewGuard()

	router := chi.NewRouter()
	router.Use(shared.CHIRouteMetaMiddleware)
	router.Use(authstd.Require(guard))

	router.Get("/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		writePrincipal(w, r)
	})
	router.Delete("/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		writePrincipal(w, r)
	})

	logger.Info("std example listening", "addr", ":8080")
	logger.Info("try curl", "command", `curl -H "Authorization: Bearer admin-token" http://127.0.0.1:8080/orders/1`)
	if err := http.ListenAndServe(":8080", router); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func writePrincipal(w http.ResponseWriter, r *http.Request) {
	principal, _ := authx.PrincipalFromContextAs[authx.Principal](r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"principal_id": principal.ID,
		"roles":        principal.Roles,
		"path":         r.URL.Path,
	})
}
