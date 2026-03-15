package udp

import (
	"errors"
	"net"

	"github.com/DaiYuANg/archgo/clientx"
	clientcodec "github.com/DaiYuANg/archgo/clientx/codec"
)

const maxUDPPacketSize = 64 * 1024

type CodecConn struct {
	conn  net.Conn
	codec clientcodec.Codec
	addr  string
}

func NewCodecConn(conn net.Conn, codec clientcodec.Codec, addr string) *CodecConn {
	return &CodecConn{
		conn:  conn,
		codec: codec,
		addr:  addr,
	}
}

func (c *CodecConn) Raw() net.Conn {
	return c.conn
}

func (c *CodecConn) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *CodecConn) WriteValue(v any) error {
	if c.codec == nil {
		return clientx.WrapErrorWithKind(
			clientx.ProtocolUDP, "encode", c.addr, clientx.ErrorKindCodec, errors.New("codec is nil"),
		)
	}
	payload, err := c.codec.Marshal(v)
	if err != nil {
		return clientx.WrapErrorWithKind(clientx.ProtocolUDP, "encode", c.addr, clientx.ErrorKindCodec, err)
	}
	if _, err := c.conn.Write(payload); err != nil {
		return clientx.WrapError(clientx.ProtocolUDP, "write", c.addr, err)
	}
	return nil
}

func (c *CodecConn) ReadValue(v any) error {
	if c.codec == nil {
		return clientx.WrapErrorWithKind(
			clientx.ProtocolUDP, "decode", c.addr, clientx.ErrorKindCodec, errors.New("codec is nil"),
		)
	}

	buf := make([]byte, maxUDPPacketSize)
	n, err := c.conn.Read(buf)
	if err != nil {
		return clientx.WrapError(clientx.ProtocolUDP, "read", c.addr, err)
	}
	if err := c.codec.Unmarshal(buf[:n], v); err != nil {
		return clientx.WrapErrorWithKind(clientx.ProtocolUDP, "decode", c.addr, clientx.ErrorKindCodec, err)
	}
	return nil
}

type CodecPacketConn struct {
	conn  net.PacketConn
	codec clientcodec.Codec
	addr  string
}

func NewCodecPacketConn(conn net.PacketConn, codec clientcodec.Codec, addr string) *CodecPacketConn {
	return &CodecPacketConn{
		conn:  conn,
		codec: codec,
		addr:  addr,
	}
}

func (c *CodecPacketConn) Raw() net.PacketConn {
	return c.conn
}

func (c *CodecPacketConn) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *CodecPacketConn) ReadValueFrom(v any) (net.Addr, error) {
	if c.codec == nil {
		return nil, clientx.WrapErrorWithKind(
			clientx.ProtocolUDP, "decode", c.addr, clientx.ErrorKindCodec, errors.New("codec is nil"),
		)
	}

	buf := make([]byte, maxUDPPacketSize)
	n, addr, err := c.conn.ReadFrom(buf)
	if err != nil {
		return nil, clientx.WrapError(clientx.ProtocolUDP, "read_from", c.addr, err)
	}
	if err := c.codec.Unmarshal(buf[:n], v); err != nil {
		return nil, clientx.WrapErrorWithKind(clientx.ProtocolUDP, "decode", c.addr, clientx.ErrorKindCodec, err)
	}
	return addr, nil
}

func (c *CodecPacketConn) WriteValueTo(v any, addr net.Addr) error {
	if c.codec == nil {
		return clientx.WrapErrorWithKind(
			clientx.ProtocolUDP, "encode", c.addr, clientx.ErrorKindCodec, errors.New("codec is nil"),
		)
	}
	payload, err := c.codec.Marshal(v)
	if err != nil {
		return clientx.WrapErrorWithKind(clientx.ProtocolUDP, "encode", c.addr, clientx.ErrorKindCodec, err)
	}

	if _, err := c.conn.WriteTo(payload, addr); err != nil {
		target := c.addr
		if addr != nil {
			target = addr.String()
		}
		return clientx.WrapError(clientx.ProtocolUDP, "write_to", target, err)
	}
	return nil
}
