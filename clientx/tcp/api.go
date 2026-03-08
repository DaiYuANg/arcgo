package tcp

import (
	"context"
	"net"

	clientcodec "github.com/DaiYuANg/arcgo/clientx/codec"
)

type Client interface {
	Dial(ctx context.Context) (net.Conn, error)
	DialCodec(ctx context.Context, codec clientcodec.Codec, framer clientcodec.Framer) (*CodecConn, error)
}

var _ Client = (*DefaultClient)(nil)
