package presenter

import (
	"go-api/internal/domain/port"
)

type RealtimeConnectionResponse struct {
	Token    string            `json:"token"`
	Channel  string            `json:"channel"`
	Channels map[string]string `json:"channels"`
	WSURL    string            `json:"wsUrl"`
}

func NewRealtimeConnectionResponse(connection port.RealtimeConnection) RealtimeConnectionResponse {
	channels := connection.Channels
	if channels == nil {
		channels = map[string]string{}
	}
	return RealtimeConnectionResponse{
		Token:    connection.Token,
		Channel:  connection.Channel,
		Channels: channels,
		WSURL:    connection.WSURL,
	}
}
