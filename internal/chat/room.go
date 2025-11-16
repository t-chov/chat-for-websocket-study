package chat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Room manages connected clients and message fan-out.
type Room struct {
	chatID string
	salt   string

	logger  *slog.Logger
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

// NewRoom builds a chat room that validates the provided chatID and salt.
func NewRoom(chatID, salt string, logger *slog.Logger) *Room {
	if logger == nil {
		logger = slog.Default()
	}

	return &Room{
		chatID:  chatID,
		salt:    salt,
		logger:  logger,
		clients: make(map[*Client]struct{}),
	}
}

// ChatID returns the configured identifier of the room.
func (r *Room) ChatID() string { return r.chatID }

// Salt returns the configured hashing salt.
func (r *Room) Salt() string { return r.salt }

// Register adds a client to the broadcast list.
func (r *Room) Register(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[c] = struct{}{}
	r.logger.Info("client joined", "name", c.Name(), "clients", len(r.clients))
}

// Unregister removes a client from the broadcast list.
func (r *Room) Unregister(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.clients[c]; ok {
		delete(r.clients, c)
		r.logger.Info("client left", "name", c.Name(), "clients", len(r.clients))
	}
}

// Broadcast delivers a message to every client except the excluded sender.
func (r *Room) Broadcast(msg OutboundMessage, exclude *Client) {
	envelope := Envelope{
		Type:      MessageTypeSystem,
		ChatID:    r.chatID,
		Body:      msg.Body,
		Timestamp: time.Now().UTC(),
	}

	if msg.System {
		envelope.Sender = "system"
	} else {
		envelope.Sender = msg.Sender
		envelope.Type = MessageTypeMessage
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		r.logger.Error("failed to marshal message", "err", err)
		return
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for client := range r.clients {
		if exclude != nil && client == exclude {
			continue
		}
		client.Enqueue(data)
	}
}

// BroadcastSystem sends a system notification to every client.
func (r *Room) BroadcastSystem(format string, args ...any) {
	r.Broadcast(OutboundMessage{Body: fmt.Sprintf(format, args...), System: true}, nil)
}
