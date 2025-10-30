package websocket_manager

type Middleware interface {
	Apply(connection *Connection) (*Connection, error)
}

type MiddlewareFunc func(connection *Connection) (*Connection, error)

func (m MiddlewareFunc) Apply(connection *Connection) (*Connection, error) {
	return m(connection)
}
