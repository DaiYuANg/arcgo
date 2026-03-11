package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/httpx/examples/shared"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
)

type doc struct {
	ID         string
	Content    string
	ETag       string
	ModifiedAt time.Time
}

type store struct {
	mu  sync.RWMutex
	doc doc
}

func newStore() *store {
	return &store{
		doc: doc{
			ID:         "1",
			Content:    "hello",
			ETag:       "v1",
			ModifiedAt: time.Now().UTC().Truncate(time.Second),
		},
	}
}

func (s *store) get() doc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.doc
}

func (s *store) update(content string) doc {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.Content = content
	s.doc.ModifiedAt = time.Now().UTC().Truncate(time.Second)
	s.doc.ETag = fmt.Sprintf("v%d", s.doc.ModifiedAt.Unix())
	return s.doc
}

type getInput struct {
	ID string `path:"id"`
	httpx.ConditionalParams
}

type getOutput struct {
	ETag         string `header:"ETag"`
	LastModified string `header:"Last-Modified"`
	Body         struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
}

type putInput struct {
	ID string `path:"id"`
	httpx.ConditionalParams
	Body struct {
		Content string `json:"content"`
	}
}

type putOutput struct {
	ETag         string `header:"ETag"`
	LastModified string `header:"Last-Modified"`
	Body         struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
}

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	defer closeLogger()

	a := std.NewWithOptions(std.Options{
		Huma: adapter.HumaOptions{
			Title:       "httpx conditional requests example",
			Version:     "1.0.0",
			Description: "If-Match / If-None-Match demo",
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
		},
	})
	s := httpx.New(httpx.WithAdapter(a))
	st := newStore()

	httpx.MustRouteWithPolicies(s, httpx.MethodGet, "/documents/{id}", func(ctx context.Context, input *getInput) (*getOutput, error) {
		current := st.get()

		out := &getOutput{}
		out.ETag = fmt.Sprintf(`"%s"`, current.ETag)
		out.LastModified = current.ModifiedAt.Format(http.TimeFormat)
		out.Body.ID = current.ID
		out.Body.Content = current.Content
		return out, nil
	}, httpx.PolicyConditionalRead[getInput, getOutput](func(ctx context.Context, input *getInput) (string, time.Time, error) {
		current := st.get()
		return current.ETag, current.ModifiedAt, nil
	}))

	httpx.MustRouteWithPolicies(s, httpx.MethodPut, "/documents/{id}", func(ctx context.Context, input *putInput) (*putOutput, error) {
		updated := st.update(input.Body.Content)
		out := &putOutput{}
		out.ETag = fmt.Sprintf(`"%s"`, updated.ETag)
		out.LastModified = updated.ModifiedAt.Format(http.TimeFormat)
		out.Body.ID = updated.ID
		out.Body.Content = updated.Content
		return out, nil
	}, httpx.PolicyConditionalWrite[putInput, putOutput](func(ctx context.Context, input *putInput) (string, time.Time, error) {
		current := st.get()
		return current.ETag, current.ModifiedAt, nil
	}))

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "conditional"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
		slog.String("first_get", fmt.Sprintf("curl -i http://localhost%s/documents/1", addr)),
		slog.String("etag_get", fmt.Sprintf("curl -i -H 'If-None-Match: \"v1\"' http://localhost%s/documents/1", addr)),
		slog.String("guarded_put", fmt.Sprintf("curl -i -X PUT -H 'Content-Type: application/json' -H 'If-Match: \"v1\"' -d '{\"content\":\"updated\"}' http://localhost%s/documents/1", addr)),
	)

	if err := s.ListenAndServe(addr); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
