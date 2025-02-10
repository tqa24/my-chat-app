package websockets

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub *Hub

	// The websocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound messages.
	Send chan []byte
	//UserID
	UserID string
}
type WebSocketMessage struct {
	Type       string `json:"type"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	GroupID    string `json:"group_id"`
	Content    string `json:"content"`
	MessageID  string `json:"message_id"`
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump(messageSaver MessageSaver) { // Changed parameter
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		var wsMessage WebSocketMessage
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue // Skip to the next iteration if unmarshaling fails
		}

		switch wsMessage.Type {
		case "new_message":
			// Use the MessageSaver interface to save the message
			//Check if receiver or group
			if wsMessage.ReceiverID != "" {
				err := messageSaver.SendMessage(wsMessage.SenderID, wsMessage.ReceiverID, "", wsMessage.Content)
				if err != nil {
					log.Printf("Error saving message: %v", err)
					continue
				}
			} else if wsMessage.GroupID != "" {
				err := messageSaver.SendMessage(wsMessage.SenderID, "", wsMessage.GroupID, wsMessage.Content)
				if err != nil {
					log.Printf("Error saving message: %v", err)
					continue
				}
			} else {
				log.Printf("Error saving message: Missing receiverID and groupID")
				continue
			}

		case "typing": // Handle typing indicator
			wsMessage.SenderID = c.UserID
			// Broadcast typing indicator to the recipient
			c.Hub.Broadcast <- message // Just forward the original message
		case "online_status":
			// Handle user coming online
			c.Hub.Broadcast <- []byte(`{"type": "online_status", "user_id": "` + c.UserID + `", "status": "online"}`)

		case "offline_status": // Handle user going offline
			// You might want to store last seen time here
			c.Hub.Broadcast <- []byte(`{"type": "offline_status", "user_id": "` + c.UserID + `"}`)

		case "read_message": // Handle message read status
			c.Hub.Broadcast <- []byte(`{"type": "read_message", "message_id": "` + wsMessage.MessageID + `", "read_by": "` + c.UserID + `"}`)

		case "join_group":
			// Add the client to the group
			c.Hub.AddClientToGroup(c.UserID, wsMessage.GroupID)
			log.Printf("Client %s joined group %s", c.UserID, wsMessage.GroupID)
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
