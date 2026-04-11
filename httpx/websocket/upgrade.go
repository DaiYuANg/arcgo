//revive:disable:file-length-limit WebSocket upgrade and connection helpers are kept together for cohesive behavior.

package websocket

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/lxzan/gws"
	"github.com/samber/oops"
)

type gwsConn struct {
	socket *gws.Conn
	opts   Options
	recv   chan Message
	errCh  chan error
	once   sync.Once
}

type eventBridge struct {
	conn *gwsConn
}

// HandlerFunc adapts a WebSocket handler to net/http.
func HandlerFunc(handler Handler, options ...Option) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := Upgrade(w, r, handler, options...); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}

// Upgrade upgrades the request to a WebSocket connection and runs handler.
func Upgrade(w http.ResponseWriter, r *http.Request, handler Handler, options ...Option) error {
	if w == nil {
		return oops.In("httpx/websocket").
			With("op", "upgrade").
			Wrapf(ErrUpgradeFailed, "response writer is nil")
	}
	if r == nil {
		return oops.In("httpx/websocket").
			With("op", "upgrade").
			Wrapf(ErrUpgradeFailed, "request is nil")
	}
	if handler == nil {
		return oops.In("httpx/websocket").
			With("op", "upgrade", "method", r.Method, "path", r.URL.Path).
			Wrapf(ErrUpgradeFailed, "nil handler")
	}
	cfg := applyOptions(options)
	bridgeConn := &gwsConn{
		opts:  cfg,
		recv:  make(chan Message, 32),
		errCh: make(chan error, 1),
	}
	upgrader := gws.NewUpgrader(&eventBridge{conn: bridgeConn}, &gws.ServerOption{
		HandshakeTimeout:   cfg.HandshakeTimeout,
		ReadMaxPayloadSize: cfg.MaxMessageSize,
		PermessageDeflate:  gws.PermessageDeflate{Enabled: cfg.EnableCompression},
		Authorize: func(req *http.Request, _ gws.SessionStorage) bool {
			if cfg.CheckOrigin == nil {
				return true
			}
			return cfg.CheckOrigin(req)
		},
	})
	socket, err := upgrader.Upgrade(w, r)
	if err != nil {
		return oops.In("httpx/websocket").
			With("op", "upgrade", "method", r.Method, "path", r.URL.Path).
			Wrapf(errors.Join(ErrUpgradeFailed, err), "upgrade websocket connection")
	}
	bridgeConn.socket = socket
	go socket.ReadLoop()

	ctx := r.Context()
	if runErr := handler(ctx, bridgeConn); runErr != nil {
		closeErr := bridgeConn.Close(1011, []byte(runErr.Error()))
		if closeErr != nil {
			return errors.Join(
				oops.In("httpx/websocket").
					With("op", "handler", "method", r.Method, "path", r.URL.Path).
					Wrapf(runErr, "websocket handler failed"),
				oops.In("httpx/websocket").
					With("op", "close", "code", 1011, "method", r.Method, "path", r.URL.Path).
					Wrapf(closeErr, "close connection after handler error"),
			)
		}
		return oops.In("httpx/websocket").
			With("op", "handler", "method", r.Method, "path", r.URL.Path).
			Wrapf(runErr, "websocket handler failed")
	}
	return nil
}

// Read reads the next WebSocket message or returns when ctx is done.
func (c *gwsConn) Read(ctx Context) (Message, error) {
	if ctx == nil {
		return Message{}, oops.In("httpx/websocket").
			With("op", "read").
			New("nil read context")
	}
	select {
	case msg, ok := <-c.recv:
		if !ok {
			return Message{}, oops.In("httpx/websocket").
				With("op", "read").
				Wrapf(ErrClosed, "websocket connection is closed")
		}
		return msg, nil
	case err := <-c.errCh:
		if err == nil {
			return Message{}, oops.In("httpx/websocket").
				With("op", "read").
				Wrapf(ErrClosed, "websocket connection is closed")
		}
		return Message{}, err
	case <-ctx.Done():
		return Message{}, oops.In("httpx/websocket").
			With("op", "read").
			Wrapf(ctx.Err(), "read canceled")
	}
}

// Write writes a typed WebSocket message to the connection.
func (c *gwsConn) Write(msg Message) (retErr error) {
	if c.socket == nil {
		return oops.In("httpx/websocket").
			With("op", "write", "message_type", msg.Type).
			Wrapf(ErrClosed, "websocket connection is closed")
	}
	cleanup, err := c.startWrite()
	if err != nil {
		return err
	}
	defer func() {
		if err := cleanup(); err != nil {
			if retErr == nil {
				retErr = err
				return
			}
			retErr = errors.Join(retErr, err)
		}
	}()

	return c.writeFrame(msg)
}

func (c *gwsConn) startWrite() (func() error, error) {
	if c.opts.WriteTimeout <= 0 {
		return noopWriteCleanup, nil
	}

	if err := c.socket.SetWriteDeadline(time.Now().Add(c.opts.WriteTimeout)); err != nil {
		return nil, oops.In("httpx/websocket").
			With("op", "write", "stage", "set_deadline", "write_timeout", c.opts.WriteTimeout).
			Wrapf(err, "set write deadline")
	}

	return func() error {
		if err := c.socket.SetWriteDeadline(time.Time{}); err != nil {
			return oops.In("httpx/websocket").
				With("op", "write", "stage", "reset_deadline").
				Wrapf(err, "reset write deadline")
		}
		return nil
	}, nil
}

func noopWriteCleanup() error {
	return nil
}

func (c *gwsConn) writeFrame(msg Message) error {
	opcode, ok := toGWSOpcode(msg.Type)
	if !ok {
		return oops.In("httpx/websocket").
			With("op", "write", "message_type", msg.Type).
			Errorf("unsupported message type: %d", msg.Type)
	}
	if msg.Type == MessageClose {
		if err := c.socket.WriteClose(1000, msg.Data); err != nil {
			return oops.In("httpx/websocket").
				With("op", "write", "message_type", msg.Type, "stage", "close_frame").
				Wrapf(err, "write close frame")
		}
		return nil
	}
	if err := c.socket.WriteMessage(opcode, msg.Data); err != nil {
		return oops.In("httpx/websocket").
			With("op", "write", "message_type", msg.Type, "payload_size", len(msg.Data)).
			Wrapf(err, "write message")
	}
	return nil
}

// Close sends a WebSocket close frame.
func (c *gwsConn) Close(code uint16, reason []byte) error {
	if c.socket == nil {
		return nil
	}
	err := c.socket.WriteClose(code, reason)
	if err != nil && !errors.Is(err, gws.ErrConnClosed) {
		return oops.In("httpx/websocket").
			With("op", "close", "code", code, "reason_size", len(reason)).
			Wrapf(err, "write close frame")
	}
	return nil
}

func (b *eventBridge) OnOpen(socket *gws.Conn) {
	b.refreshReadDeadlines(socket)
}

func (b *eventBridge) OnClose(_ *gws.Conn, err error) {
	b.conn.once.Do(func() {
		if err != nil {
			b.conn.errCh <- err
		}
		close(b.conn.recv)
		close(b.conn.errCh)
	})
}

func (b *eventBridge) OnPing(socket *gws.Conn, payload []byte) {
	b.refreshReadDeadlines(socket)
	if err := socket.WritePong(payload); err != nil {
		b.conn.reportError(oops.In("httpx/websocket").
			With("op", "pong", "payload_size", len(payload)).
			Wrapf(err, "write pong frame"))
	}
}

func (b *eventBridge) OnPong(socket *gws.Conn, _ []byte) {
	b.refreshIdleDeadline(socket)
}

func (b *eventBridge) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer func() {
		if err := message.Close(); err != nil {
			b.conn.reportError(oops.In("httpx/websocket").
				With("op", "read", "stage", "close_message").
				Wrapf(err, "close message"))
		}
	}()
	msgType, ok := fromGWSOpcode(message.Opcode)
	if !ok {
		return
	}
	b.refreshReadDeadlines(socket)
	payload := append([]byte(nil), message.Bytes()...)
	b.conn.recv <- Message{Type: msgType, Data: payload}
}

func toGWSOpcode(t MessageType) (gws.Opcode, bool) {
	switch t {
	case MessageText:
		return gws.OpcodeText, true
	case MessageBinary:
		return gws.OpcodeBinary, true
	case MessagePing:
		return gws.OpcodePing, true
	case MessagePong:
		return gws.OpcodePong, true
	case MessageClose:
		return gws.OpcodeCloseConnection, true
	default:
		return 0, false
	}
}

func fromGWSOpcode(op gws.Opcode) (MessageType, bool) {
	switch op {
	case gws.OpcodeText:
		return MessageText, true
	case gws.OpcodeBinary:
		return MessageBinary, true
	case gws.OpcodePing:
		return MessagePing, true
	case gws.OpcodePong:
		return MessagePong, true
	case gws.OpcodeCloseConnection:
		return MessageClose, true
	case gws.OpcodeContinuation:
		return 0, false
	default:
		return 0, false
	}
}

func (c *gwsConn) reportError(err error) {
	if err == nil {
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			return
		}
	}()

	select {
	case c.errCh <- err:
	default:
	}
}

func (b *eventBridge) refreshReadDeadlines(socket *gws.Conn) {
	b.refreshIdleDeadline(socket)
	if b.conn.opts.ReadTimeout > 0 {
		if err := socket.SetReadDeadline(time.Now().Add(b.conn.opts.ReadTimeout)); err != nil {
			b.conn.reportError(oops.In("httpx/websocket").
				With("op", "read", "stage", "set_deadline", "read_timeout", b.conn.opts.ReadTimeout).
				Wrapf(err, "set read deadline"))
		}
	}
}

func (b *eventBridge) refreshIdleDeadline(socket *gws.Conn) {
	if b.conn.opts.IdleTimeout > 0 {
		if err := socket.SetDeadline(time.Now().Add(b.conn.opts.IdleTimeout)); err != nil {
			b.conn.reportError(oops.In("httpx/websocket").
				With("op", "read", "stage", "set_idle_deadline", "idle_timeout", b.conn.opts.IdleTimeout).
				Wrapf(err, "set idle deadline"))
		}
	}
}
