package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// TestWritePumpSendsIndividualFrames verifies that when multiple messages
// are queued in the send channel, each one is delivered as its own
// WebSocket frame (i.e. independently parseable JSON).
func TestWritePumpSendsIndividualFrames(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Set up a test HTTP server that upgrades to WebSocket and runs WritePump.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		client := NewClient(hub, conn, 1)
		// Pre-load multiple messages before WritePump starts draining.
		client.send <- []byte(`{"type":"output","content":"hello"}`)
		client.send <- []byte(`{"type":"output","content":"world"}`)
		client.send <- []byte(`{"type":"output","content":"three"}`)
		go client.WritePump()
	}))
	defer server.Close()

	// Connect a client WebSocket.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer clientConn.Close()

	// Read three separate frames, each should be valid JSON.
	received := make([]map[string]string, 0, 3)
	clientConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for i := 0; i < 3; i++ {
		_, msg, err := clientConn.ReadMessage()
		if err != nil {
			t.Fatalf("read message %d: %v", i, err)
		}
		var parsed map[string]string
		if err := json.Unmarshal(msg, &parsed); err != nil {
			t.Fatalf("frame %d is not valid JSON: %q, err: %v", i, string(msg), err)
		}
		received = append(received, parsed)
	}

	expected := []string{"hello", "world", "three"}
	for i, want := range expected {
		if got := received[i]["content"]; got != want {
			t.Errorf("frame %d: got content %q, want %q", i, got, want)
		}
	}
}

// TestWritePumpClosesOnChannelClose verifies WritePump sends a close
// message when the send channel is closed.
func TestWritePumpClosesOnChannelClose(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	done := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		client := NewClient(hub, conn, 1)
		// Close the send channel immediately so WritePump exits.
		close(client.send)
		client.WritePump()
		close(done)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer clientConn.Close()

	// WritePump should exit promptly after channel close.
	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("WritePump did not exit after channel close")
	}
}
