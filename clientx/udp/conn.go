package udp

import (
	"net"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
)

type timeoutConn struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
	addr         string
	hooks        []clientx.Hook
}

func (c *timeoutConn) Read(b []byte) (int, error) {
	if c.readTimeout > 0 {
		_ = c.SetReadDeadline(time.Now().Add(c.readTimeout))
	}
	start := time.Now()
	n, err := c.Conn.Read(b)
	if err != nil {
		wrappedErr := clientx.WrapError(clientx.ProtocolUDP, "read", c.addr, err)
		clientx.EmitIO(c.hooks, clientx.IOEvent{
			Protocol: clientx.ProtocolUDP,
			Op:       "read",
			Addr:     c.addr,
			Bytes:    n,
			Duration: time.Since(start),
			Err:      wrappedErr,
		})
		return n, wrappedErr
	}
	clientx.EmitIO(c.hooks, clientx.IOEvent{
		Protocol: clientx.ProtocolUDP,
		Op:       "read",
		Addr:     c.addr,
		Bytes:    n,
		Duration: time.Since(start),
	})
	return n, nil
}

func (c *timeoutConn) Write(b []byte) (int, error) {
	if c.writeTimeout > 0 {
		_ = c.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}
	start := time.Now()
	n, err := c.Conn.Write(b)
	if err != nil {
		wrappedErr := clientx.WrapError(clientx.ProtocolUDP, "write", c.addr, err)
		clientx.EmitIO(c.hooks, clientx.IOEvent{
			Protocol: clientx.ProtocolUDP,
			Op:       "write",
			Addr:     c.addr,
			Bytes:    n,
			Duration: time.Since(start),
			Err:      wrappedErr,
		})
		return n, wrappedErr
	}
	clientx.EmitIO(c.hooks, clientx.IOEvent{
		Protocol: clientx.ProtocolUDP,
		Op:       "write",
		Addr:     c.addr,
		Bytes:    n,
		Duration: time.Since(start),
	})
	return n, nil
}

type timeoutPacketConn struct {
	net.PacketConn
	readTimeout  time.Duration
	writeTimeout time.Duration
	addr         string
	hooks        []clientx.Hook
}

func (c *timeoutPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	if c.readTimeout > 0 {
		_ = c.SetReadDeadline(time.Now().Add(c.readTimeout))
	}
	start := time.Now()
	n, addr, err := c.PacketConn.ReadFrom(b)
	if err != nil {
		wrappedErr := clientx.WrapError(clientx.ProtocolUDP, "read_from", c.addr, err)
		clientx.EmitIO(c.hooks, clientx.IOEvent{
			Protocol: clientx.ProtocolUDP,
			Op:       "read_from",
			Addr:     c.addr,
			Bytes:    n,
			Duration: time.Since(start),
			Err:      wrappedErr,
		})
		return n, addr, wrappedErr
	}
	clientx.EmitIO(c.hooks, clientx.IOEvent{
		Protocol: clientx.ProtocolUDP,
		Op:       "read_from",
		Addr:     c.addr,
		Bytes:    n,
		Duration: time.Since(start),
	})
	return n, addr, nil
}

func (c *timeoutPacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	if c.writeTimeout > 0 {
		_ = c.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}
	start := time.Now()
	n, err := c.PacketConn.WriteTo(b, addr)
	if err != nil {
		target := c.addr
		if addr != nil {
			target = addr.String()
		}
		wrappedErr := clientx.WrapError(clientx.ProtocolUDP, "write_to", target, err)
		clientx.EmitIO(c.hooks, clientx.IOEvent{
			Protocol: clientx.ProtocolUDP,
			Op:       "write_to",
			Addr:     target,
			Bytes:    n,
			Duration: time.Since(start),
			Err:      wrappedErr,
		})
		return n, wrappedErr
	}
	target := c.addr
	if addr != nil {
		target = addr.String()
	}
	clientx.EmitIO(c.hooks, clientx.IOEvent{
		Protocol: clientx.ProtocolUDP,
		Op:       "write_to",
		Addr:     target,
		Bytes:    n,
		Duration: time.Since(start),
	})
	return n, nil
}
