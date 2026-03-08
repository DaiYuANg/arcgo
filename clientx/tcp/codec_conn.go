package tcp

import (
	"errors"
	"net"

	"github.com/DaiYuANg/arcgo/clientx"
	clientcodec "github.com/DaiYuANg/arcgo/clientx/codec"
)

type CodecConn struct {
	conn   net.Conn
	codec  clientcodec.Codec
	framer clientcodec.Framer
	addr   string
}

func NewCodecConn(conn net.Conn, codec clientcodec.Codec, framer clientcodec.Framer, addr string) *CodecConn {
	return &CodecConn{
		conn:   conn,
		codec:  codec,
		framer: framer,
		addr:   addr,
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
	if c.codec == nil || c.framer == nil {
		return clientx.WrapErrorWithKind(
			clientx.ProtocolTCP, "encode", c.addr, clientx.ErrorKindCodec, errors.New("codec/framer is nil"),
		)
	}

	payload, err := c.codec.Marshal(v)
	if err != nil {
		return clientx.WrapErrorWithKind(clientx.ProtocolTCP, "encode", c.addr, clientx.ErrorKindCodec, err)
	}
	if err := c.framer.WriteFrame(c.conn, payload); err != nil {
		return clientx.WrapError(clientx.ProtocolTCP, "write_frame", c.addr, err)
	}
	return nil
}

func (c *CodecConn) ReadValue(v any) error {
	if c.codec == nil || c.framer == nil {
		return clientx.WrapErrorWithKind(
			clientx.ProtocolTCP, "decode", c.addr, clientx.ErrorKindCodec, errors.New("codec/framer is nil"),
		)
	}

	frame, err := c.framer.ReadFrame(c.conn)
	if err != nil {
		return clientx.WrapError(clientx.ProtocolTCP, "read_frame", c.addr, err)
	}
	if err := c.codec.Unmarshal(frame, v); err != nil {
		return clientx.WrapErrorWithKind(clientx.ProtocolTCP, "decode", c.addr, clientx.ErrorKindCodec, err)
	}
	return nil
}
