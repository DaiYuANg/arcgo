package udp

import (
	"context"

	"github.com/DaiYuANg/archgo/clientx"
	clientcodec "github.com/DaiYuANg/archgo/clientx/codec"
)

type Client interface {
	clientx.Closer
	clientx.Dialer
	clientx.PacketListener
	DialCodec(ctx context.Context, codec clientcodec.Codec) (*CodecConn, error)
	ListenPacketCodec(ctx context.Context, codec clientcodec.Codec) (*CodecPacketConn, error)
}

var _ Client = (*DefaultClient)(nil)
var _ clientx.PacketListener = (*DefaultClient)(nil)
