package main

const (
	TypeChat             = "chat"
	TypeUserConnected    = "user_connected"
	TypeUserDisconnected = "user_disconnected"
	TypeClientList       = "client_list"
)

type MessageRequest struct {
	Message string `json:"message"`
	To      string `json:"to"`
}

type MessageResponse struct {
	Message string `json:"message"`
	From    string `json:"from"`
}

type Message struct {
	Data interface{} `json:"data"`
	Type string      `json:"type"`
}
