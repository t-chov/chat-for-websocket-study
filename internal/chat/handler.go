package chat

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocketHandler upgrades HTTP requests and coordinates client lifecycles.
type WebSocketHandler struct {
	room     *Room
	upgrader websocket.Upgrader
}

// NewWebSocketHandler wires the chat room to an HTTP handler.
func NewWebSocketHandler(room *Room) *WebSocketHandler {
	return &WebSocketHandler{
		room: room,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// ServeHTTP upgrades the connection and blocks while the client session is active.
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(conn, h.room, h.room.logger)
	client.Serve(r.Context())
}
