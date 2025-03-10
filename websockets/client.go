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
	maxMessageSize = 8192 // 8KB
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
	Type             string `json:"type"`
	SenderID         string `json:"sender_id"`
	ReceiverID       string `json:"receiver_id"`
	GroupID          string `json:"group_id"`
	Content          string `json:"content"`
	MessageID        string `json:"message_id"`
	ReplyToMessageID string `json:"reply_to_message_id"`
	Emoji            string `json:"emoji"`
	Status           string `json:"status"`
	// File fields
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	FileType     string `json:"file_type"`
	FileSize     int64  `json:"file_size"`
	FileChecksum string `json:"checksum"`
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump(messageSaver MessageSaver) {
	defer func() {
		// When client disconnects, send offline status before unregistering
		offlineMsg := []byte(`{"type": "offline_status", "user_id": "` + c.UserID + `"}`)
		c.Hub.Broadcast <- offlineMsg
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
			// *DO NOT* call chatService.SendMessage here!
			// The consumer handles saving and broadcasting.
			log.Printf("Received new_message via WebSocket.  This should NOT happen now.")

		case "typing": // Handle typing indicator
			wsMessage.SenderID = c.UserID
			// Broadcast typing indicator to the recipient
			c.Hub.Broadcast <- message // Just forward the original message
		case "online_status":
			// Handle user coming online
			statusMsg := []byte(`{"type": "online_status", "user_id": "` + c.UserID + `"}`)
			c.Hub.Broadcast <- statusMsg

		case "offline_status": // Handle user going offline
			// You might want to store last seen time here
			statusMsg := []byte(`{"type": "offline_status", "user_id": "` + c.UserID + `"}`)
			c.Hub.Broadcast <- statusMsg

		case "read_message": // Handle message read status
			c.Hub.Broadcast <- []byte(`{"type": "read_message", "message_id": "` + wsMessage.MessageID + `", "read_by": "` + c.UserID + `"}`)

		case "join_group":
			// Add the client to the group
			c.Hub.AddClientToGroup(c.UserID, wsMessage.GroupID)
			log.Printf("Client %s joined group %s", c.UserID, wsMessage.GroupID)
		case "reaction":
			// Handle adding reaction
			if messageSaver, ok := messageSaver.(interface {
				AddReaction(messageID, userID, emoji string) error
			}); ok {
				err := messageSaver.AddReaction(wsMessage.MessageID, c.UserID, wsMessage.Emoji)
				if err != nil {
					log.Printf("Error adding reaction: %v", err)
					continue
				}
				// Broadcast the reaction update
				//c.Hub.Broadcast <- message // Remove this.  Backend handles it.
			}

		case "remove_reaction":
			// Handle removing reaction
			if messageSaver, ok := messageSaver.(interface {
				RemoveReaction(messageID, userID, emoji string) error
			}); ok {
				err := messageSaver.RemoveReaction(wsMessage.MessageID, c.UserID, wsMessage.Emoji)
				if err != nil {
					log.Printf("Error removing reaction: %v", err)
					continue
				}
				// Broadcast the reaction removal
				//c.Hub.Broadcast <- message // Remove this.  Backend handles it.
			}
		case "message_status":
			if messageSaver, ok := messageSaver.(interface {
				UpdateMessageStatus(messageID string, status string) error
			}); ok {
				err := messageSaver.UpdateMessageStatus(wsMessage.MessageID, wsMessage.Status)
				if err != nil {
					log.Printf("Error updating message status: %v", err)
					continue
				}
				// Broadcast the status update
				c.Hub.Broadcast <- message
			}
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
