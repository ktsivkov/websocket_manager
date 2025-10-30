package websocket_manager

import "context"

type MessageReader interface {
	Read(ctx context.Context, payload []byte) error
}

type MessageReaderFunc func(ctx context.Context, payload []byte) error

func (h MessageReaderFunc) Read(ctx context.Context, payload []byte) error {
	return h(ctx, payload)
}

type MessageWriter interface {
	Write(ctx context.Context, ch chan<- []byte) error
}

type MessageWriterFunc func(ctx context.Context, ch chan<- []byte) error

func (h MessageWriterFunc) Write(ctx context.Context, ch chan<- []byte) error {
	return h(ctx, ch)
}

type ConnectionEventHandler interface {
	On(ctx context.Context)
}

type ConnectionEventHandlerFunc func(ctx context.Context)

func (h ConnectionEventHandlerFunc) On(ctx context.Context) {
	h(ctx)
}
