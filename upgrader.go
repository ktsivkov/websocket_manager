package websocket_manager

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error)
}

func Upgrade(
	w http.ResponseWriter,
	r *http.Request,
	upgrader Upgrader,
	responseHeader http.Header,
	clientCreator SocketCreator,
	conf *Config,
) (*Worker, error) {
	conn, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return nil, err
	}

	client, err := clientCreator.Create()
	if err != nil {
		return nil, err
	}

	return newWorker(conn, client, conf), nil
}
