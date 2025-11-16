package chat

import "time"

const (
	MessageTypeJoin    = "join"
	MessageTypeToken   = "token"
	MessageTypeMessage = "message"
	MessageTypeSystem  = "system"
	MessageTypeError   = "error"
)

// Envelope describes the JSON payload exchanged over the WebSocket.
type Envelope struct {
	Type      string    `json:"type"`
	ChatID    string    `json:"chatId,omitempty"`
	Name      string    `json:"name,omitempty"`
	Token     string    `json:"token,omitempty"`
	Body      string    `json:"body,omitempty"`
	Sender    string    `json:"sender,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// OutboundMessage describes a message broadcast to the room.
type OutboundMessage struct {
	Body   string
	Sender string
	System bool
}
