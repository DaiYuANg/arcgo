package main

import (
	"context"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"time"

	clienthttp "github.com/DaiYuANg/archgo/clientx/http"
	"github.com/DaiYuANg/archgo/clientx/preset"
)

func main() {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.URL.Path != "/ping" {
			w.WriteHeader(stdhttp.StatusNotFound)
			return
		}
		w.WriteHeader(stdhttp.StatusOK)
		_, _ = w.Write([]byte("pong"))
	}))
	defer srv.Close()

	client, err := preset.NewEdgeHTTP(
		clienthttp.Config{BaseURL: srv.URL},
		preset.WithEdgeHTTPDisableRetry(),
		preset.WithEdgeHTTPTimeout(2*time.Second),
		preset.WithEdgeHTTPTimeoutGuard(1500*time.Millisecond),
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.Execute(context.Background(), nil, stdhttp.MethodGet, "/ping")
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("status=%d body=%q\n", resp.StatusCode(), string(body))
}
