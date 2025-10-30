package websocket_manager

type ControlMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var errMsgMalformed = ControlMessage{
	Code:    "REQUEST_MALFORMED",
	Message: "Request malformed",
}
