package main

type MessageRequest struct {
	Message string `json:"message"`
	To      string `json:"to"`
}

type MessageResponse struct {
	Message string `json:"message"`
	From    string `json:"from"`
}
