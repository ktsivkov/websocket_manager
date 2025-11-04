package websocket_manager

import (
	"errors"
	"net"
)

var (
	ErrWriteTimeoutExceeded = errors.New("write timeout exceeded")
	ErrPongTimeoutExceeded  = errors.New("pong timeout exceeded")
	ErrWorkerAlreadyRun     = errors.New("worker has already run")
	ErrWriterChannelClosed  = errors.New("writer channel closed")
	ErrCloseMessageSent     = errors.New("close message sent")
	ErrCloseMessageReceived = errors.New("close message received")
	ErrPingMessage          = errors.New("failed to write ping message")
	ErrFailedToWrite        = errors.New("failed to write")
	ErrConnectionClosed     = errors.New("connection closed")
	ErrFailedToRead         = errors.New("failed to read")
)

func isConnectionClosedError(err error) bool {
	return errors.Is(err, net.ErrClosed)
}

func isTimeoutExceededError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
