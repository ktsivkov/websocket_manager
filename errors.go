package websocket_manager

import "errors"

var (
	ErrTimeoutExceeded      = errors.New("timeout exceeded")
	ErrConnectionClosed     = errors.New("connection closed")
	ErrAlreadyClosed        = errors.New("already closed")
	ErrFailedToMarshal      = errors.New("failed to marshal")
	ErrMiddlewareFailed     = errors.New("middleware failed")
	ErrConnectionCloseError = errors.New("connection close error")
)
