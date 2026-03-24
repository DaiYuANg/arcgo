package websocket

import "errors"

var (
	ErrClosed        = errors.New("httpx/websocket: connection closed")
	ErrUpgradeFailed = errors.New("httpx/websocket: upgrade failed")
)

type MessageType uint8

const (
	MessageText MessageType = iota + 1
	MessageBinary
	MessagePing
	MessagePong
	MessageClose
)

type Message struct {
	Type MessageType
	Data []byte
}

type Handler func(Context, Conn) error

type Conn interface {
	Read(Context) (Message, error)
	Write(Message) error
	Close(code uint16, reason []byte) error
}
