package chat

import (
	"io"
	"log/slog"
	"testing"
)

func TestRoomBroadcast(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	room := NewRoom("1234564", "salt", logger)

	alice := &Client{name: "alice", send: make(chan []byte, 1), logger: logger}
	bob := &Client{name: "bob", send: make(chan []byte, 1), logger: logger}

	room.Register(alice)
	room.Register(bob)

	room.Broadcast(OutboundMessage{Body: "hello", Sender: "alice"}, alice)

	select {
	case msg := <-bob.send:
		if len(msg) == 0 {
			t.Fatalf("expected payload")
		}
	default:
		t.Fatalf("bob should receive a message")
	}

	select {
	case <-alice.send:
		t.Fatalf("sender should be excluded")
	default:
	}
}
