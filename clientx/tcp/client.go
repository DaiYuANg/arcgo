package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"time"

	"github.com/DaiYuANg/archgo/clientx"
	clientcodec "github.com/DaiYuANg/archgo/clientx/codec"
)

type DefaultClient struct {
	cfg      Config
	hooks    []clientx.Hook
	policies []clientx.Policy
}

func New(cfg Config, opts ...Option) (Client, error) {
	normalized, err := cfg.NormalizeAndValidate()
	if err != nil {
		return nil, err
	}

	c := &DefaultClient{cfg: normalized}
	clientx.Apply(c, opts...)
	return c, nil
}

func (c *DefaultClient) Close() error {
	return nil
}

func (c *DefaultClient) Dial(ctx context.Context) (net.Conn, error) {
	network := c.cfg.Network
	operation := clientx.Operation{
		Protocol: clientx.ProtocolTCP,
		Kind:     clientx.OperationKindDial,
		Op:       "dial",
		Network:  network,
		Addr:     c.cfg.Address,
	}

	return clientx.InvokeWithPolicies(ctx, operation, func(execCtx context.Context) (net.Conn, error) {
		start := time.Now()
		dialer := &net.Dialer{
			Timeout:   c.cfg.DialTimeout,
			KeepAlive: c.cfg.KeepAlive,
		}

		if c.cfg.TLS.Enabled {
			tlsDialer := &tls.Dialer{
				NetDialer: dialer,
				Config: &tls.Config{
					InsecureSkipVerify: c.cfg.TLS.InsecureSkipVerify,
					ServerName:         c.cfg.TLS.ServerName,
				},
			}
			conn, err := tlsDialer.DialContext(execCtx, network, c.cfg.Address)
			if err != nil {
				wrappedErr := clientx.WrapError(clientx.ProtocolTCP, "dial", c.cfg.Address, err)
				clientx.EmitDial(c.hooks, clientx.DialEvent{
					Protocol: clientx.ProtocolTCP,
					Op:       "dial",
					Network:  network,
					Addr:     c.cfg.Address,
					Duration: time.Since(start),
					Err:      wrappedErr,
				})
				return nil, wrappedErr
			}
			clientx.EmitDial(c.hooks, clientx.DialEvent{
				Protocol: clientx.ProtocolTCP,
				Op:       "dial",
				Network:  network,
				Addr:     c.cfg.Address,
				Duration: time.Since(start),
			})
			return wrapConn(conn, c.cfg, c.hooks), nil
		}

		conn, err := dialer.DialContext(execCtx, network, c.cfg.Address)
		if err != nil {
			wrappedErr := clientx.WrapError(clientx.ProtocolTCP, "dial", c.cfg.Address, err)
			clientx.EmitDial(c.hooks, clientx.DialEvent{
				Protocol: clientx.ProtocolTCP,
				Op:       "dial",
				Network:  network,
				Addr:     c.cfg.Address,
				Duration: time.Since(start),
				Err:      wrappedErr,
			})
			return nil, wrappedErr
		}
		clientx.EmitDial(c.hooks, clientx.DialEvent{
			Protocol: clientx.ProtocolTCP,
			Op:       "dial",
			Network:  network,
			Addr:     c.cfg.Address,
			Duration: time.Since(start),
		})
		return wrapConn(conn, c.cfg, c.hooks), nil
	}, c.policies...)
}

func (c *DefaultClient) DialCodec(ctx context.Context, codec clientcodec.Codec, framer clientcodec.Framer) (*CodecConn, error) {
	if codec == nil {
		return nil, clientx.WrapErrorWithKind(
			clientx.ProtocolTCP, "dial_codec", c.cfg.Address, clientx.ErrorKindCodec, errors.New("codec is nil"),
		)
	}
	if framer == nil {
		return nil, clientx.WrapErrorWithKind(
			clientx.ProtocolTCP, "dial_codec", c.cfg.Address, clientx.ErrorKindCodec, errors.New("framer is nil"),
		)
	}

	conn, err := c.Dial(ctx)
	if err != nil {
		return nil, err
	}
	return NewCodecConn(conn, codec, framer, c.cfg.Address), nil
}

func wrapConn(conn net.Conn, cfg Config, hooks []clientx.Hook) net.Conn {
	return &timeoutConn{
		Conn:         conn,
		readTimeout:  cfg.ReadTimeout,
		writeTimeout: cfg.WriteTimeout,
		addr:         cfg.Address,
		hooks:        hooks,
	}
}
