package tcp

import (
	"context"

	"github.com/DaiYuANg/archgo/clientx"
	clientcodec "github.com/DaiYuANg/archgo/clientx/codec"
)

type Client interface {
	clientx.Closer
	clientx.Dialer
	DialCodec(ctx context.Context, codec clientcodec.Codec, framer clientcodec.Framer) (*CodecConn, error)
}

var _ Client = (*DefaultClient)(nil)
var _ clientx.Dialer = (*DefaultClient)(nil)
