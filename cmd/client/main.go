package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"

	"github.com/t-chov/websocket-with-ai/internal/chat"
)

func main() {
	var (
		serverURL = flag.String("server", "ws://localhost:28080/ws", "websocket server URL")
		chatID    = flag.String("chat-id", "1234564", "chat identifier")
		name      = flag.String("name", "", "display name (required)")
		timeout   = flag.Duration("timeout", 10*time.Second, "connection timeout")
	)
	flag.Parse()

	if strings.TrimSpace(*name) == "" {
		fmt.Fprintln(os.Stderr, "--name is required")
		os.Exit(2)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	d := websocket.Dialer{HandshakeTimeout: *timeout}
	conn, _, err := d.DialContext(ctx, *serverURL, nil)
	if err != nil {
		logger.Error("dial failed", "url", *serverURL, "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := newWSClient(conn, *chatID, strings.TrimSpace(*name), logger)

	if err := client.join(); err != nil {
		logger.Error("join failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go client.readLoop(ctx, cancel)
	go client.inputLoop(ctx)

	<-ctx.Done()
	logger.Info("shutting down client")
}

type wsClient struct {
	conn         wsConn
	chatID       string
	name         string
	token        atomic.Value
	logger       *slog.Logger
	isCloseError func(error) bool
}

type wsConn interface {
	WriteJSON(v interface{}) error
	ReadJSON(v interface{}) error
	Close() error
}

func newWSClient(conn wsConn, chatID, name string, logger *slog.Logger) *wsClient {
	if logger == nil {
		logger = slog.Default()
	}
	client := &wsClient{
		conn:   conn,
		chatID: chatID,
		name:   name,
		logger: logger,
	}
	client.isCloseError = func(err error) bool {
		return websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure)
	}
	return client
}

func (c *wsClient) join() error {
	joinEnvelope := chat.Envelope{
		Type:   chat.MessageTypeJoin,
		ChatID: c.chatID,
		Name:   c.name,
	}
	if err := c.conn.WriteJSON(joinEnvelope); err != nil {
		return fmt.Errorf("send join: %w", err)
	}
	c.logger.Info("sent join", "chatID", c.chatID, "name", c.name)
	return nil
}

func (c *wsClient) readLoop(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var env chat.Envelope
		if err := c.conn.ReadJSON(&env); err != nil {
			if c.isCloseError != nil && c.isCloseError(err) {
				c.logger.Info("server closed connection", "reason", err)
			} else {
				c.logger.Error("read error", "err", err)
			}
			return
		}

		c.handleEnvelope(env)
	}
}

func (c *wsClient) handleEnvelope(env chat.Envelope) {
	switch env.Type {
	case chat.MessageTypeToken:
		if env.Token == "" {
			c.logger.Warn("token envelope missing token")
			return
		}
		c.token.Store(env.Token)
		fmt.Printf("[system] issued token %s\n", env.Token)
	case chat.MessageTypeSystem:
		fmt.Printf("[%s] %s\n", env.Sender, env.Body)
	case chat.MessageTypeMessage:
		ts := env.Timestamp.Format(time.Kitchen)
		if ts == "" {
			ts = time.Now().Format(time.Kitchen)
		}
		fmt.Printf("[%s][%s] %s\n", env.Sender, ts, env.Body)
	case chat.MessageTypeError:
		fmt.Fprintf(os.Stderr, "[error] %s\n", env.Error)
	default:
		fmt.Printf("[unknown] %+v\n", env)
	}
}

func (c *wsClient) inputLoop(ctx context.Context) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type messages and press Enter to send. Ctrl+C to exit.")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "input error: %v\n", err)
			}
			return
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if err := c.sendMessage(line); err != nil {
			fmt.Fprintf(os.Stderr, "send error: %v\n", err)
			return
		}
	}
}

func (c *wsClient) sendMessage(body string) error {
	token, ok := c.token.Load().(string)
	if !ok || token == "" {
		return errors.New("token not yet issued by server")
	}

	env := chat.Envelope{
		Type:   chat.MessageTypeMessage,
		Token:  token,
		Body:   body,
		Sender: c.name,
	}

	if err := c.conn.WriteJSON(env); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}
