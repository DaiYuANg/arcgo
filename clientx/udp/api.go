package udp

import (
	"context"
	"net"

	clientcodec "github.com/DaiYuANg/arcgo/clientx/codec"
)

type Client interface {
	Dial(ctx context.Context) (net.Conn, error)
	ListenPacket(ctx context.Context) (net.PacketConn, error)
	DialCodec(ctx context.Context, codec clientcodec.Codec) (*CodecConn, error)
	ListenPacketCodec(ctx context.Context, codec clientcodec.Codec) (*CodecPacketConn, error)
}

var _ Client = (*DefaultClient)(nil)
