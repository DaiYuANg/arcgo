// Package redis_test verifies the public redis adapter behavior.
package redis_test

import (
	"errors"
	"testing"

	"github.com/DaiYuANg/arcgo/kvx"
	redisadapter "github.com/DaiYuANg/arcgo/kvx/adapter/redis"
)

func TestNew_ValidatesClientOptions(t *testing.T) {
	tests := []struct {
		name string
		opts kvx.ClientOptions
		want error
	}{
		{
			name: "empty addrs",
			opts: kvx.ClientOptions{},
			want: kvx.ErrInvalidClientOptions,
		},
		{
			name: "tls unsupported",
			opts: kvx.ClientOptions{Addrs: []string{"127.0.0.1:6379"}, UseTLS: true},
			want: kvx.ErrUnsupportedOption,
		},
		{
			name: "sentinel unsupported",
			opts: kvx.ClientOptions{Addrs: []string{"127.0.0.1:6379"}, MasterName: "mymaster"},
			want: kvx.ErrUnsupportedOption,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := redisadapter.New(tt.opts)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}
