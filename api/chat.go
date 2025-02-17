package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"my-chat-app/services"
	"my-chat-app/websockets"
	"net/http"
	"os"
	"path/filepath"
)

const (
	MaxFileSize = 25 * 1024 * 1024 // 25 MB
	UploadDir   = "./uploads"      // IMPORTANT:  Create this directory in your project root
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

	go client.WritePump()                                       // Handle sending messages to the client
	go client.ReadPump(h.chatService.(websockets.MessageSaver)) // Pass the chatService here and fix.
}

// --- File Upload Handler ---
func (h *ChatHandler) UploadFile(c *gin.Context) {
	// 1. Check file size (server-side)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxFileSize)
	err := c.Request.ParseMultipartForm(MaxFileSize)
	if err != nil {
		if err.Error() == "http: request body too large" {
			// Log detailed error information.
			log.Printf("UploadFile: File too large: %v, Error: %v", c.Request.ContentLength, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds the limit (25MB)"})
			return
		}
	}
	// 2. Get the file from the request
	file, header, err := c.Request.FormFile("file") // "file" is the name of the form field
	if err != nil {
		//Log error
		log.Printf("UploadFile: Error retrieving file from form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not retrieve file"})
		return
	}
	defer file.Close()

	// 3. Validate file type (optional, but recommended) - Example: Allow only images
	// Check file extension
	ext := filepath.Ext(header.Filename)
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".pdf":  true, // Add PDF for document support
		".txt":  true,
		// Add more as needed...
	}
	if _, ok := allowedExts[ext]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type"})
		return
	}

	// 4. Generate a unique filename.  Prevent overwrites, avoid spaces/special chars.
	uniqueID := uuid.New()
	filename := fmt.Sprintf("%s%s", uniqueID.String(), filepath.Ext(header.Filename))
	filePath := filepath.Join(UploadDir, filename)

	// 5. Create the uploads directory if it doesn't exist
	if _, err := os.Stat(UploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(UploadDir, 0755); err != nil { // 0755 is a common permission
			log.Printf("UploadFile: Error creating upload directory: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create upload directory"})
			return
		}
	}
	// 6. Save the file to the uploads directory.
	outFile, err := os.Create(filePath)
	if err != nil {
		log.Printf("UploadFile: Error creating file on disk: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save file"})
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		log.Printf("UploadFile: Error writing file to disk: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save file"})
		return
	}

	// 7. Get file information
	fileInfo, err := outFile.Stat()
	if err != nil {
		log.Printf("UploadFile: Error getting file info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get file information"})
		return
	}
	fileSize := fileInfo.Size()
	// 8. Return the filename and path to the client
	c.JSON(http.StatusOK, gin.H{
		"filename": filename,                           // The unique filename we saved as
		"filepath": filepath.Join("uploads", filename), // Relative path for access
		"filetype": header.Header.Get("Content-Type"),  // Get the content type
		"filesize": fileSize,                           // Return file size
	})
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	var wsMessage websockets.WebSocketMessage
	if err := c.ShouldBindJSON(&wsMessage); err != nil {
		log.Printf("SendMessage: Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	log.Printf("Received via WebSocket: %+v", wsMessage) // Log the entire struct

	// --- Basic validation ---
	if wsMessage.SenderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sender_id"})
		return
	}
	if (wsMessage.ReceiverID == "" && wsMessage.GroupID == "") || (wsMessage.ReceiverID != "" && wsMessage.GroupID != "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Specify either receiver_id or group_id, not both"})
		return
	}
	if wsMessage.FileName == "" && wsMessage.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content is required"})
		return
	}

	// --- Call the ChatService ---
	// Correctly use the fields from wsMessage
	err := h.chatService.SendMessage(
		wsMessage.SenderID,
		wsMessage.ReceiverID,
		wsMessage.GroupID,
		wsMessage.Content,
		wsMessage.ReplyToMessageID,
		wsMessage.FileName, // Use wsMessage.FileName, etc.
		wsMessage.FilePath,
		wsMessage.FileType,
		wsMessage.FileSize,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}

// GetGroupConversation handles retrieving the conversation history for a group.
func (h *ChatHandler) GetGroupConversation(c *gin.Context) {
	groupID := c.Param("id")

	page := c.DefaultQuery("page", "1")          // Default to page 1
	pageSize := c.DefaultQuery("pageSize", "10") // Default page size

	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "groupID parameters are required"})
		return
	}

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
