package tcp

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
		wrappedErr := clientx.WrapError(clientx.ProtocolTCP, "read", c.addr, err)
		clientx.EmitIO(c.hooks, clientx.IOEvent{
			Protocol: clientx.ProtocolTCP,
			Op:       "read",
			Addr:     c.addr,
			Bytes:    n,
			Duration: time.Since(start),
			Err:      wrappedErr,
		})
		return n, wrappedErr
	}
	clientx.EmitIO(c.hooks, clientx.IOEvent{
		Protocol: clientx.ProtocolTCP,
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
		wrappedErr := clientx.WrapError(clientx.ProtocolTCP, "write", c.addr, err)
		clientx.EmitIO(c.hooks, clientx.IOEvent{
			Protocol: clientx.ProtocolTCP,
			Op:       "write",
			Addr:     c.addr,
			Bytes:    n,
			Duration: time.Since(start),
			Err:      wrappedErr,
		})
		return n, wrappedErr
	}
	clientx.EmitIO(c.hooks, clientx.IOEvent{
		Protocol: clientx.ProtocolTCP,
		Op:       "write",
		Addr:     c.addr,
		Bytes:    n,
		Duration: time.Since(start),
	})
	return n, nil
}
