package udp

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	clientcodec "github.com/DaiYuANg/arcgo/clientx/codec"
)

type DefaultClient struct {
	cfg   Config
	hooks []clientx.Hook
}

func New(cfg Config, opts ...Option) Client {
	c := &DefaultClient{cfg: cfg}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *DefaultClient) Dial(ctx context.Context) (net.Conn, error) {
	network := c.cfg.Network
	if network == "" {
		network = "udp"
	}
	start := time.Now()

	dialer := &net.Dialer{
		Timeout: c.cfg.DialTimeout,
	}

	conn, err := dialer.DialContext(ctx, network, c.cfg.Address)
	if err != nil {
		wrappedErr := clientx.WrapError(clientx.ProtocolUDP, "dial", c.cfg.Address, err)
		clientx.EmitDial(c.hooks, clientx.DialEvent{
			Protocol: clientx.ProtocolUDP,
			Op:       "dial",
			Network:  network,
			Addr:     c.cfg.Address,
			Duration: time.Since(start),
			Err:      wrappedErr,
		})
		return nil, wrappedErr
	}
	clientx.EmitDial(c.hooks, clientx.DialEvent{
		Protocol: clientx.ProtocolUDP,
		Op:       "dial",
		Network:  network,
		Addr:     c.cfg.Address,
		Duration: time.Since(start),
	})

	return &timeoutConn{
		Conn:         conn,
		readTimeout:  c.cfg.ReadTimeout,
		writeTimeout: c.cfg.WriteTimeout,
		addr:         c.cfg.Address,
		hooks:        c.hooks,
	}, nil
}

func (c *DefaultClient) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	network := c.cfg.Network
	if network == "" {
		network = "udp"
	}
	start := time.Now()

	lc := &net.ListenConfig{}
	conn, err := lc.ListenPacket(ctx, network, c.cfg.Address)
	if err != nil {
		wrappedErr := clientx.WrapError(clientx.ProtocolUDP, "listen", c.cfg.Address, err)
		clientx.EmitDial(c.hooks, clientx.DialEvent{
			Protocol: clientx.ProtocolUDP,
			Op:       "listen",
			Network:  network,
			Addr:     c.cfg.Address,
			Duration: time.Since(start),
			Err:      wrappedErr,
		})
		return nil, wrappedErr
	}
	clientx.EmitDial(c.hooks, clientx.DialEvent{
		Protocol: clientx.ProtocolUDP,
		Op:       "listen",
		Network:  network,
		Addr:     c.cfg.Address,
		Duration: time.Since(start),
	})

	return &timeoutPacketConn{
		PacketConn:   conn,
		readTimeout:  c.cfg.ReadTimeout,
		writeTimeout: c.cfg.WriteTimeout,
		addr:         c.cfg.Address,
		hooks:        c.hooks,
	}, nil
}

func (c *DefaultClient) DialCodec(ctx context.Context, codec clientcodec.Codec) (*CodecConn, error) {
	if codec == nil {
		return nil, clientx.WrapErrorWithKind(
			clientx.ProtocolUDP, "dial_codec", c.cfg.Address, clientx.ErrorKindCodec, errors.New("codec is nil"),
		)
	}

	conn, err := c.Dial(ctx)
	if err != nil {
		return nil, err
	}
	return NewCodecConn(conn, codec, c.cfg.Address), nil
}

func (c *DefaultClient) ListenPacketCodec(ctx context.Context, codec clientcodec.Codec) (*CodecPacketConn, error) {
	if codec == nil {
		return nil, clientx.WrapErrorWithKind(
			clientx.ProtocolUDP, "listen_codec", c.cfg.Address, clientx.ErrorKindCodec, errors.New("codec is nil"),
		)
	}

	packetConn, err := c.ListenPacket(ctx)
	if err != nil {
		return nil, err
	}
	return NewCodecPacketConn(packetConn, codec, c.cfg.Address), nil
}
