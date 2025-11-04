package websocket_manager

import (
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type Message interface {
	// Write writes to the connection.
	// Returns ErrConnectionClosed if the connection is closed.
	// Returns ErrWriteTimeoutExceeded if the writing timed out.
	Write(conn *websocket.Conn) error
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
	msg *websocket.PreparedMessage
	typ int
}

func (m *message) Write(conn *websocket.Conn) error {
	if err := conn.WritePreparedMessage(m.msg); err != nil {
		if isConnectionClosedError(err) {
			return fmt.Errorf("%w: %w", ErrConnectionClosed, err)
		}
		return err
	}

	return nil
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
	data    []byte
	typ     int
	timeout time.Duration
}

func (m *controlMessage) Type() int {
	return m.typ
}

func (m *controlMessage) Write(conn *websocket.Conn) error {
	if err := conn.WriteControl(m.typ, m.data, time.Now().Add(m.timeout)); err != nil {
		if isConnectionClosedError(err) {
			return fmt.Errorf("%w: %w", ErrConnectionClosed, err)
		}
		if isTimeoutExceededError(err) {
			return fmt.Errorf("%w: %w", ErrWriteTimeoutExceeded, err)
		}
		return err
	}
	return nil
}

type ClientCloseMessage struct {
	Code int
	Text string
}

func clientCloseMessageFromError(err error) *ClientCloseMessage {
	val := &websocket.CloseError{}
	if errors.As(err, &val) {
		return &ClientCloseMessage{
			Code: val.Code,
			Text: val.Text,
		}
	}

	return nil
}
