package main

var ControlMessageUsernameTaken ControlMessage = ControlMessage{
	Code:    "USERNAME_ALREADY_TAKEN",
	Message: "This username is already taken. Please choose another one.",
}

type ControlMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
