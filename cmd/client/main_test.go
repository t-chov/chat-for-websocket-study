package main

import (
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/t-chov/websocket-with-ai/internal/chat"
)

type mockConn struct {
	writes   []interface{}
	writeErr error
}

func (m *mockConn) WriteJSON(v interface{}) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.writes = append(m.writes, v)
	return nil
}

func (m *mockConn) ReadJSON(v interface{}) error { return nil }
func (m *mockConn) Close() error                 { return nil }

func discardOutput(t *testing.T) func() {
	t.Helper()
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open dev null: %v", err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	os.Stdout = devNull
	os.Stderr = devNull
	return func() {
		os.Stdout = stdout
		os.Stderr = stderr
		devNull.Close()
	}
}

func TestJoinSendsEnvelope(t *testing.T) {
	conn := &mockConn{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := newWSClient(conn, "1234564", "alice", logger)

	if err := client.join(); err != nil {
		t.Fatalf("join returned error: %v", err)
	}

	if len(conn.writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(conn.writes))
	}

	env, ok := conn.writes[0].(chat.Envelope)
	if !ok {
		t.Fatalf("expected chat.Envelope, got %T", conn.writes[0])
	}

	if env.Type != chat.MessageTypeJoin || env.ChatID != "1234564" || env.Name != "alice" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

func TestSendMessageRequiresToken(t *testing.T) {
	conn := &mockConn{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := newWSClient(conn, "1234564", "bob", logger)

	if err := client.sendMessage("hi"); err == nil {
		t.Fatalf("expected error when token missing")
	}
}

func TestSendMessageWritesEnvelope(t *testing.T) {
	conn := &mockConn{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := newWSClient(conn, "1234564", "carol", logger)
	client.token.Store("abc123")

	if err := client.sendMessage("hello"); err != nil {
		t.Fatalf("sendMessage returned error: %v", err)
	}

	if len(conn.writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(conn.writes))
	}

	env := conn.writes[0].(chat.Envelope)
	if env.Type != chat.MessageTypeMessage || env.Token != "abc123" || env.Body != "hello" || env.Sender != "carol" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

func TestHandleEnvelopeStoresToken(t *testing.T) {
	cleanup := discardOutput(t)
	defer cleanup()

	conn := &mockConn{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := newWSClient(conn, "1234564", "dave", logger)

	client.handleEnvelope(chat.Envelope{Type: chat.MessageTypeToken, Token: "token-value"})

	token, _ := client.token.Load().(string)
	if token != "token-value" {
		t.Fatalf("expected token to be stored, got %s", token)
	}
}
