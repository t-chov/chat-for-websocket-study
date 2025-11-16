package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8192
)

// Client represents a single WebSocket participant.
type Client struct {
	conn   *websocket.Conn
	room   *Room
	name   string
	token  string
	send   chan []byte
	logger *slog.Logger
}

// NewClient builds a client around the WebSocket connection.
func NewClient(conn *websocket.Conn, room *Room, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		conn:   conn,
		room:   room,
		send:   make(chan []byte, 64),
		logger: logger,
	}
}

// Name returns the client's display name.
func (c *Client) Name() string { return c.name }

// Serve handles the lifecycle of the client connection.
func (c *Client) Serve(ctx context.Context) {
	var joined bool

	defer func() {
		if joined {
			c.room.Unregister(c)
			c.room.Broadcast(OutboundMessage{Body: fmt.Sprintf("%s left", c.name), System: true}, c)
		}
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go c.writeLoop(ctx)

	if err := c.handshake(); err != nil {
		c.logger.Warn("handshake failed", "err", err)
		c.sendError(err)
		return
	}

	joined = true
	c.room.Register(c)
	c.sendEnvelope(Envelope{Type: MessageTypeToken, Token: c.token, ChatID: c.room.ChatID()})
	c.room.BroadcastSystem("%s joined", c.name)

	c.readLoop(ctx)
}

func (c *Client) handshake() error {
	var env Envelope
	if err := c.conn.ReadJSON(&env); err != nil {
		return fmt.Errorf("read join envelope: %w", err)
	}

	if env.Type != MessageTypeJoin {
		return errors.New("expected join message")
	}

	if strings.TrimSpace(env.ChatID) != c.room.ChatID() {
		return errors.New("unknown chat ID")
	}

	name := strings.TrimSpace(env.Name)
	if name == "" {
		return errors.New("name is required")
	}

	c.name = name
	c.token = GenerateToken(c.room.ChatID(), name, c.room.Salt())
	return nil
}

func (c *Client) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var env Envelope
		if err := c.conn.ReadJSON(&env); err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Info("client disconnected", "name", c.name, "reason", err)
			} else {
				c.logger.Warn("read error", "name", c.name, "err", err)
			}
			return
		}

		switch env.Type {
		case MessageTypeMessage:
			c.handleChatMessage(env)
		default:
			c.sendError(fmt.Errorf("unsupported message type %s", env.Type))
		}
	}
}

func (c *Client) handleChatMessage(env Envelope) {
	body := strings.TrimSpace(env.Body)
	if body == "" {
		c.sendError(errors.New("message body required"))
		return
	}

	if env.Token == "" {
		c.sendError(errors.New("token required"))
		return
	}

	if !ValidateToken(c.room.ChatID(), c.name, c.room.Salt(), env.Token) {
		c.sendError(errors.New("invalid token"))
		return
	}

	c.room.Broadcast(OutboundMessage{Body: body, Sender: c.name}, c)
}

func (c *Client) writeLoop(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-c.send:
			if !ok {
				_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.write(message); err != nil {
				c.logger.Warn("write error", "name", c.name, "err", err)
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Warn("ping failed", "name", c.name, "err", err)
				return
			}
		}
	}
}

func (c *Client) write(payload []byte) error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

// Enqueue buffers a message for async delivery, dropping when the buffer is full.
func (c *Client) Enqueue(payload []byte) {
	select {
	case c.send <- payload:
	default:
		c.logger.Warn("dropping message", "name", c.name)
	}
}

func (c *Client) sendEnvelope(env Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		c.logger.Error("marshal envelope failed", "err", err)
		return
	}
	c.Enqueue(data)
}

func (c *Client) sendError(err error) {
	c.sendEnvelope(Envelope{Type: MessageTypeError, Error: err.Error()})
}
