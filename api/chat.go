package api

import (
	"fmt"
	"log"
	"my-chat-app/services"
	"my-chat-app/websockets"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatHandler struct {
	chatService services.ChatService
	hub         *websockets.Hub // Inject the WebSocket hub
}

func NewChatHandler(chatService services.ChatService, hub *websockets.Hub) *ChatHandler {
	return &ChatHandler{chatService, hub}
}

// GetConversation handles retrieving the conversation history between two users.
func (h *ChatHandler) GetConversation(c *gin.Context) {
	user1ID := c.Query("user1")
	user2ID := c.Query("user2")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("pageSize", "10")

	if user1ID == "" || user2ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both user1 and user2 parameters are required"})
		return
	}

	messages, err := h.chatService.GetConversation(user1ID, user2ID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve conversation"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development.  In production, you *MUST* restrict this.
		return true
	},
}

func (h *ChatHandler) WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}
	userID := c.Query("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UserID is required"})
		return
	}

	client := &websockets.Client{
		Hub:    h.hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}

	client.Hub.Register <- client

	go client.WritePump()             // Handle sending messages to the client
	go client.ReadPump(h.chatService) // Pass the chatService here!
}
func (h *ChatHandler) SendMessage(c *gin.Context) {
	// Extract message details from request
	senderID := c.PostForm("sender_id")
	receiverID := c.PostForm("receiver_id")
	content := c.PostForm("content")
	fmt.Println(senderID, receiverID, content) // Keep this for debugging
	// Validate the data
	if senderID == "" || receiverID == "" || content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	// Call the ChatService to send the message
	err := h.chatService.SendMessage(senderID, receiverID, content) // Corrected line
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Return a success response
	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}
