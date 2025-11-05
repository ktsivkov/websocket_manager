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
	Write(conn *websocket.Conn, timeout time.Duration) error
	Type() int
}

func TextMessage(payload string) Message {
	msg, err := websocket.NewPreparedMessage(websocket.TextMessage, []byte(payload))
	if err != nil {
		panic(fmt.Errorf("failed to prepare text message: %w (%s)", err, payload))
	}

	return &message{
		typ: websocket.TextMessage,
		msg: msg,
	}
}

func BinaryMessage(payload []byte) Message {
	msg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, payload)
	if err != nil {
		panic(fmt.Errorf("failed to prepare binary message: %w (%s)", err, payload))
	}

	return &message{
		typ: websocket.BinaryMessage,
		msg: msg,
	}
}

func PingMessage(payload []byte) Message {
	msg, err := websocket.NewPreparedMessage(websocket.PingMessage, payload)
	if err != nil {
		panic(fmt.Errorf("failed to prepare ping message: %w (%s)", err, payload))
	}

	return &message{
		typ: websocket.PingMessage,
		msg: msg,
	}
}

// CloseMessage creates a new Message that writes a close message to the connection.
// Check valid status codes at https://pkg.go.dev/github.com/gorilla/websocket#pkg-constants.
// Panics if it cannot prepare the message.
func CloseMessage(status int, payload string) Message {
	msg, err := websocket.NewPreparedMessage(websocket.CloseMessage, websocket.FormatCloseMessage(status, payload))
	if err != nil {
		panic(fmt.Errorf("failed to prepare close message: %w (status=%d, payload=%s)", err, status, payload))
	}

	return &message{
		typ: websocket.CloseMessage,
		msg: msg,
	}
}

type message struct {
	msg *websocket.PreparedMessage
	typ int
}

func (m *message) Write(conn *websocket.Conn, timeout time.Duration) error {
	if timeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	}
	if err := conn.WritePreparedMessage(m.msg); err != nil {
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

func (m *message) Type() int {
	return m.typ
}

type ClientCloseMessage struct {
	Text string
	Code int
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
