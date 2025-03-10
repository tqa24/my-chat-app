package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
	"io"
	"log"
	"my-chat-app/models"
	"my-chat-app/services"
	"my-chat-app/utils"
	"my-chat-app/websockets"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	MaxFileSize = 25 * 1024 * 1024 // 25 MB
	UploadDir   = "./uploads"      // Create directory in project root
)

type ChatHandler struct {
	chatService services.ChatService
	hub         *websockets.Hub // Inject the WebSocket hub
	db          *gorm.DB
	amqpChannel *amqp.Channel
	jwtService  services.JWTService
}

func NewChatHandler(chatService services.ChatService, hub *websockets.Hub, db *gorm.DB, amqpChannel *amqp.Channel, jwtService services.JWTService) *ChatHandler {
	return &ChatHandler{chatService, hub, db, amqpChannel, jwtService}
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
	}, // Allow all origins for development
}

func (h *ChatHandler) WebSocketHandler(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		utils.RespondWithError(c, http.StatusUnauthorized, "Token is required")
		return
	}

	// Validate the token
	jwtToken, err := h.jwtService.ValidateToken(token)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	// Extract user ID from token
	userID, err := h.jwtService.GetUserIDFromToken(jwtToken)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "Failed to extract user information")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &websockets.Client{
		Hub:    h.hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}
	client.Hub.Register <- client

	go client.WritePump()
	go client.ReadPump(h.chatService.(websockets.MessageSaver))
}

// --- File Upload Handler ---
func (h *ChatHandler) UploadFile(c *gin.Context) {
	// Set maximum file size
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxFileSize)
	err := c.Request.ParseMultipartForm(MaxFileSize)
	if err != nil {
		if err.Error() == "http: request body too large" {
			log.Printf("UploadFile: File too large: %v, Error: %v", c.Request.ContentLength, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds the limit (25MB)"})
			return
		}
		log.Printf("UploadFile: Error parsing multipart form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get the file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Printf("UploadFile: Error retrieving file from form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not retrieve file"})
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".pdf": true, ".txt": true,
		".zip": true, ".doc": true, ".docx": true, ".ppt": true, ".pptx": true, ".xls": true,
		".xlsx": true, ".csv": true, ".mp4": true, ".mp3": true, ".wav": true, ".flac": true,
		".ogg": true, ".avi": true, ".mov": true, ".wmv": true, ".webm": true, ".mkv": true,
		".svg": true, ".json": true, ".xml": true, ".html": true, ".css": true, ".js": true,
		".go": true, ".java": true, ".py": true,
	}
	if _, ok := allowedExts[ext]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type"})
		return
	}

	// Calculate SHA-256 checksum
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		log.Printf("UploadFile: Error calculating checksum: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process file"})
		return
	}
	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Check for existing file with same checksum
	var existingFile models.Message
	result := h.db.Where("file_checksum = ?", checksum).First(&existingFile)
	if result.Error == nil {
		// File exists - return existing file information
		c.JSON(http.StatusOK, gin.H{
			"filename":  existingFile.FileName,
			"filepath":  existingFile.FilePath,
			"filetype":  existingFile.FileType,
			"filesize":  existingFile.FileSize,
			"checksum":  existingFile.FileChecksum,
			"duplicate": true,
			"message":   "File already exists",
		})
		return
	} else if result.Error != gorm.ErrRecordNotFound {
		// Database error
		log.Printf("UploadFile: Database error checking for existing file: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Rewind file for saving
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		log.Printf("UploadFile: Error rewinding file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process file"})
		return
	}

	// Generate unique filename
	uniqueID := uuid.New()
	filename := fmt.Sprintf("%s%s", uniqueID.String(), filepath.Ext(header.Filename))
	filePath := filepath.Join(UploadDir, filename)

	// Create uploads directory if it doesn't exist
	if _, err := os.Stat(UploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(UploadDir, 0755); err != nil {
			log.Printf("UploadFile: Error creating upload directory: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create upload directory"})
			return
		}
	}

	// Create the file
	outFile, err := os.Create(filePath)
	if err != nil {
		log.Printf("UploadFile: Error creating file on disk: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save file"})
		return
	}
	defer outFile.Close()

	// Copy the file
	if _, err := io.Copy(outFile, file); err != nil {
		log.Printf("UploadFile: Error writing file to disk: %v", err)
		// Clean up the partially written file
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save file"})
		return
	}

	// Get file info for size
	fileInfo, err := outFile.Stat()
	if err != nil {
		log.Printf("UploadFile: Error getting file info: %v", err)
		// Clean up
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get file information"})
		return
	}

	// Format the file path for URL
	urlFilePath := strings.ReplaceAll(filepath.Join("uploads", filename), "\\", "/")

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"filename":  filename,
		"filepath":  urlFilePath,
		"filetype":  header.Header.Get("Content-Type"),
		"filesize":  fileInfo.Size(),
		"checksum":  checksum,
		"duplicate": false,
		"message":   "File uploaded successfully",
	})
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	var wsMessage websockets.WebSocketMessage
	if err := c.ShouldBindJSON(&wsMessage); err != nil {
		log.Printf("SendMessage: Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Basic validation
	if wsMessage.SenderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sender_id"})
		return
	}
	if (wsMessage.ReceiverID == "" && wsMessage.GroupID == "") ||
		(wsMessage.ReceiverID != "" && wsMessage.GroupID != "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Specify either receiver_id or group_id, not both"})
		return
	}
	if wsMessage.FileName == "" && wsMessage.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content or file is required"})
		return
	}

	// Check content size
	const maxContentSize = 8192 // 8KB
	if len(wsMessage.Content) > maxContentSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "Message content exceeds maximum size limit",
		})
		return
	}

	// File existence check
	if wsMessage.FileName != "" {
		filePath := filepath.Join(UploadDir, wsMessage.FileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File does not exist"})
			return
		}
	}
	// Convert the WebSocketMessage to JSON
	msgBytes, err := json.Marshal(wsMessage)
	if err != nil {
		log.Printf("Error marshaling message to JSON: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process message"})
		return
	}
	// Publish the message to RabbitMQ
	err = h.amqpChannel.Publish(
		"",           // exchange
		"chat_queue", // routing key (queue name)
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msgBytes,
		})
	if err != nil {
		log.Printf("Error publishing message to RabbitMQ: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message queued for sending"}) // Indicate successful queuing

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
