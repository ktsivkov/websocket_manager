package websocket_manager

import (
	"time"

	"github.com/gorilla/websocket"
)

type Message interface {
	// Write writes to the connection.
	// Returns ErrConnectionClosed if the connection is closed.
	// Returns ErrTimeoutExceeded if the writing timed out.
	Write(conn *Connection) error
	Type() int
}

func TextMessage(payload string) (Message, error) {
	msg, err := websocket.NewPreparedMessage(websocket.TextMessage, []byte(payload))
	if err != nil {
		return nil, err
	}

	return &message{
		typ: websocket.TextMessage,
		msg: msg,
	}, nil
}

func BinaryMessage(payload []byte) (Message, error) {
	msg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, payload)
	if err != nil {
		return nil, err
	}

	return &message{
		typ: websocket.BinaryMessage,
		msg: msg,
	}, nil
}

type message struct {
	typ int
	msg *websocket.PreparedMessage
}

func (m *message) Write(conn *Connection) error {
	return conn.WritePreparedMessage(m.msg)
}

func (m *message) Type() int {
	return m.typ
}

// CloseMessage creates a new Message that writes a close message to the connection.
// Check valid status codes at https://pkg.go.dev/github.com/gorilla/websocket#pkg-constants.
func CloseMessage(status int, payload string, timeout time.Duration) Message {
	data := websocket.FormatCloseMessage(status, payload)
	return &controlMessage{
		typ:     websocket.CloseMessage,
		data:    data,
		timeout: timeout,
	}
}

type controlMessage struct {
	typ     int
	data    []byte
	timeout time.Duration
}

func (m *controlMessage) Type() int {
	return m.typ
}

func (m *controlMessage) Write(conn *Connection) error {
	return conn.WriteControl(m.typ, m.data, m.timeout)
}
