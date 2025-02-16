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
	// Change here
	page := c.DefaultQuery("page", "1")          // Default to page 1
	pageSize := c.DefaultQuery("pageSize", "10") // Default page size of 10

	if user1ID == "" || user2ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both user1 and user2 parameters are required"})
		return
	}

	//Change here
	messages, total, err := h.chatService.GetConversation(user1ID, user2ID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve conversation"})
		return
	}
	// Return messages and total count
	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"total":    total,    // Total number of messages
		"page":     page,     // Current page
		"pageSize": pageSize, // Page size
	})
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
	// Extract message details from request (you might get this from a JSON body)
	senderID := c.PostForm("sender_id")
	receiverID := c.PostForm("receiver_id")
	groupID := c.PostForm("group_id") // Get group_id
	content := c.PostForm("content")
	replyToMessageID := c.PostForm("reply_to_message_id") // Get reply_to_message_id
	fmt.Println(senderID, receiverID, content)
	// Validate the data
	if senderID == "" || content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	if receiverID == "" && groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ReceiverID or groupID required"})
		return
	}
	// Check if both receiverID and groupID are provided
	if receiverID != "" && groupID != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot specify both receiverID and groupID"})
		return
	}

	// Call the ChatService to send the message
	err := h.chatService.SendMessage(senderID, receiverID, groupID, content, replyToMessageID) // Updated call
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Return a success response
	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}

// GetGroupConversation handles retrieving the conversation history for a group.
func (h *ChatHandler) GetGroupConversation(c *gin.Context) {
	groupID := c.Param("id")
	// Change here
	page := c.DefaultQuery("page", "1")          // Default to page 1
	pageSize := c.DefaultQuery("pageSize", "10") // Default page size

	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "groupID parameters are required"})
		return
	}

	// Change here
	messages, total, err := h.chatService.GetGroupConversation(groupID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve conversation"})
		return
	}

	// Return messages and total count
	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"total":    total,    // Total number of messages
		"page":     page,     // Current page
		"pageSize": pageSize, // Page size
	})
}

// AddReaction handles adding a reaction to a message
func (h *ChatHandler) AddReaction(c *gin.Context) {
	messageID := c.Param("id") // Get message ID
	var req struct {
		UserID   string `json:"user_id"`
		Reaction string `json:"reaction"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	userID := req.UserID     // User ID
	reaction := req.Reaction // Reaction string
	if messageID == "" || userID == "" || reaction == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing parameters"})
		return
	}

	err := h.chatService.AddReaction(messageID, userID, reaction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reaction added"})
}

// RemoveReaction handles removing a reaction from a message
func (h *ChatHandler) RemoveReaction(c *gin.Context) {
	messageID := c.Param("id")
	var req struct {
		UserID   string `json:"user_id"`
		Reaction string `json:"reaction"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	userID := req.UserID     // User ID
	reaction := req.Reaction // Reaction string

	if messageID == "" || userID == "" || reaction == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing parameters"})
		return
	}

	err := h.chatService.RemoveReaction(messageID, userID, reaction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reaction removed"})
}
