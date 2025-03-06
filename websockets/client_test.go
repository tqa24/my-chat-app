package websockets

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMessageSizeLimits(t *testing.T) {
	// Create test messages of different sizes
	smallMsg := make([]byte, 4*1024)  // 4KB
	mediumMsg := make([]byte, 8*1024) // 8KB (at the limit)
	largeMsg := make([]byte, 10*1024) // 10KB (exceeds limit)

	// Fill with cryptographically secure random data
	_, err := rand.Read(smallMsg)
	if err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}
	_, err = rand.Read(mediumMsg)
	if err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}
	_, err = rand.Read(largeMsg)
	if err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}

	// Convert to base64 strings (simulating JSON content)
	smallStr := base64.StdEncoding.EncodeToString(smallMsg)
	mediumStr := base64.StdEncoding.EncodeToString(mediumMsg)
	largeStr := base64.StdEncoding.EncodeToString(largeMsg)

	// Create WebSocket test server with a channel to signal when connections are closed
	connectionClosed := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Upgrade failed: %v", err)
		}

		// Set the read limit to match your application's limit
		conn.SetReadLimit(maxMessageSize)

		// Set a handler for when the connection closes
		conn.SetCloseHandler(func(code int, text string) error {
			connectionClosed <- true
			return nil
		})

		defer conn.Close()

		// Read messages until connection closes
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				// Connection closed or error reading
				connectionClosed <- true
				break
			}
		}
	}))
	defer server.Close()

	// Convert http to ws
	url := "ws" + strings.TrimPrefix(server.URL, "http")

	// Test small message (should succeed)
	t.Run("Small message (4KB)", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("Dial failed: %v", err)
		}
		defer conn.Close()

		err = conn.WriteMessage(websocket.TextMessage, []byte(smallStr))
		if err != nil {
			t.Errorf("Failed to send small message: %v", err)
		}

		// Wait a moment to see if the connection was closed
		select {
		case <-connectionClosed:
			t.Errorf("Connection closed unexpectedly for small message")
		case <-time.After(100 * time.Millisecond):
			// No closure, this is expected
		}
	})

	// Test medium message (at the limit, should succeed)
	t.Run("Medium message (8KB)", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("Dial failed: %v", err)
		}
		defer conn.Close()

		err = conn.WriteMessage(websocket.TextMessage, []byte(mediumStr))
		if err != nil {
			t.Errorf("Failed to send medium message: %v", err)
		}

		// Wait a moment to see if the connection was closed
		select {
		case <-connectionClosed:
			t.Errorf("Connection closed unexpectedly for medium message")
		case <-time.After(100 * time.Millisecond):
			// No closure, this is expected
		}
	})

	// Test large message (exceeds limit, should fail)
	t.Run("Large message (10KB)", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("Dial failed: %v", err)
		}
		defer conn.Close()

		// Send the large message
		err = conn.WriteMessage(websocket.TextMessage, []byte(largeStr))

		// The write itself might not fail immediately
		// But the connection should be closed by the server
		select {
		case <-connectionClosed:
			// Connection was closed as expected
		case <-time.After(500 * time.Millisecond):
			t.Errorf("Large message should have caused connection closure but didn't")

			// Try to send another message to verify connection state
			err = conn.WriteMessage(websocket.TextMessage, []byte("test"))
			if err == nil {
				t.Errorf("Connection should be closed but is still accepting messages")
			}
		}
	})
	t.Logf("Small message encoded size: %d bytes", len(smallStr))
	t.Logf("Medium message encoded size: %d bytes", len(mediumStr))
	t.Logf("Large message encoded size: %d bytes", len(largeStr))
}
