package websocket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lxzan/gws"
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

func HandlerFunc(handler Handler, options ...Option) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := Upgrade(w, r, handler, options...); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}

func Upgrade(w http.ResponseWriter, r *http.Request, handler Handler, options ...Option) error {
	if handler == nil {
		return fmt.Errorf("%w: nil handler", ErrUpgradeFailed)
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
		return fmt.Errorf("%w: %w", ErrUpgradeFailed, err)
	}
	bridgeConn.socket = socket
	go socket.ReadLoop()

	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if runErr := handler(ctx, bridgeConn); runErr != nil {
		_ = bridgeConn.Close(1011, []byte(runErr.Error()))
		return runErr
	}
	return nil
}

func (c *gwsConn) Read(ctx Context) (Message, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case msg, ok := <-c.recv:
		if !ok {
			return Message{}, ErrClosed
		}
		return msg, nil
	case err := <-c.errCh:
		if err == nil {
			return Message{}, ErrClosed
		}
		return Message{}, err
	case <-ctx.Done():
		return Message{}, ctx.Err()
	}
}

func (c *gwsConn) Write(msg Message) error {
	if c.socket == nil {
		return ErrClosed
	}
	if c.opts.WriteTimeout > 0 {
		_ = c.socket.SetWriteDeadline(time.Now().Add(c.opts.WriteTimeout))
		defer func() { _ = c.socket.SetWriteDeadline(time.Time{}) }()
	}
	opcode, ok := toGWSOpcode(msg.Type)
	if !ok {
		return fmt.Errorf("httpx/websocket: unsupported message type: %d", msg.Type)
	}
	if msg.Type == MessageClose {
		return c.socket.WriteClose(1000, msg.Data)
	}
	return c.socket.WriteMessage(opcode, msg.Data)
}

func (c *gwsConn) Close(code uint16, reason []byte) error {
	if c.socket == nil {
		return nil
	}
	err := c.socket.WriteClose(code, reason)
	if err != nil && !errors.Is(err, gws.ErrConnClosed) {
		return err
	}
	return nil
}

func (b *eventBridge) OnOpen(socket *gws.Conn) {
	if b.conn.opts.IdleTimeout > 0 {
		_ = socket.SetDeadline(time.Now().Add(b.conn.opts.IdleTimeout))
	}
	if b.conn.opts.ReadTimeout > 0 {
		_ = socket.SetReadDeadline(time.Now().Add(b.conn.opts.ReadTimeout))
	}
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
	if b.conn.opts.IdleTimeout > 0 {
		_ = socket.SetDeadline(time.Now().Add(b.conn.opts.IdleTimeout))
	}
	if b.conn.opts.ReadTimeout > 0 {
		_ = socket.SetReadDeadline(time.Now().Add(b.conn.opts.ReadTimeout))
	}
	_ = socket.WritePong(payload)
}

func (b *eventBridge) OnPong(socket *gws.Conn, _ []byte) {
	if b.conn.opts.IdleTimeout > 0 {
		_ = socket.SetDeadline(time.Now().Add(b.conn.opts.IdleTimeout))
	}
}

func (b *eventBridge) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer message.Close()
	msgType, ok := fromGWSOpcode(message.Opcode)
	if !ok {
		return
	}
	if b.conn.opts.IdleTimeout > 0 {
		_ = socket.SetDeadline(time.Now().Add(b.conn.opts.IdleTimeout))
	}
	if b.conn.opts.ReadTimeout > 0 {
		_ = socket.SetReadDeadline(time.Now().Add(b.conn.opts.ReadTimeout))
	}
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
	default:
		return 0, false
	}
}
