package main

import (
	"fmt"
	"log"
	"my-chat-app/api"
	"my-chat-app/config"
	"my-chat-app/repositories"
	"my-chat-app/services"
	"my-chat-app/websockets"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	config.LoadConfig()
	// Connect to PostgreSQL
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.AppConfig.DBHost, config.AppConfig.DBUser, config.AppConfig.DBPassword, config.AppConfig.DBName, config.AppConfig.DBPort)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	messageRepo := repositories.NewMessageRepository(db)
	groupRepo := repositories.NewGroupRepository(db)

	// Initialize WebSocket hub
	hub := websockets.NewHub()
	go hub.Run() // Run the hub in a separate goroutine

	// Initialize services
	authService := services.NewAuthService(userRepo)
	chatService := services.NewChatService(messageRepo, groupRepo, userRepo, hub)
	groupService := services.NewGroupService(groupRepo, userRepo, hub)

	// Initialize handlers
	authHandler := api.NewAuthHandler(authService, userRepo)
	chatHandler := api.NewChatHandler(chatService, hub, db)
	groupHandler := api.NewGroupHandler(groupService)
	// Initialize Gin router
	r := gin.Default()
	//CORS
	r.Use(CORSMiddleware())
	// Routes
	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
	r.POST("/logout", authHandler.Logout)
	r.GET("/profile", authHandler.Profile)
	r.GET("/ws", chatHandler.WebSocketHandler)
	r.GET("/messages", chatHandler.GetConversation) // For get conversation
	r.POST("/messages", chatHandler.SendMessage)    // For send message
	r.GET("/users", authHandler.GetAllUsers)
	// Group routes
	r.POST("/groups", groupHandler.CreateGroup)                     // Create a new group
	r.GET("/groups/:id", groupHandler.GetGroup)                     // Get group details
	r.POST("/groups/:id/join", groupHandler.JoinGroup)              // Join a group
	r.POST("/groups/join-by-code", groupHandler.JoinGroupByCode)    // Join group by code
	r.POST("/groups/:id/leave", groupHandler.LeaveGroup)            // Leave a group
	r.GET("/users/:id/groups", groupHandler.ListGroupsForUser)      // List groups for a user
	r.GET("/groups", groupHandler.GetAllGroups)                     // Get all groups
	r.GET("/groups/:id/messages", chatHandler.GetGroupConversation) // For get group conversation
	r.POST("/messages/:id/react", chatHandler.AddReaction)          // NEW: Add reaction
	r.DELETE("/messages/:id/react", chatHandler.RemoveReaction)     // NEW: Remove reaction
	// *** NEW: File Upload Route ***
	r.POST("/upload", chatHandler.UploadFile)
	// *** NEW: Serve uploaded files statically ***
	r.Static("/uploads", "./uploads")

	// AI route
	r.POST("/ai", chatHandler.PrivateAIHandler) // NEW: Add the /ai route
	
	// Start the server
	log.Printf("Server listening on port %s", config.AppConfig.AppPort)
	log.Fatal(http.ListenAndServe(":"+config.AppConfig.AppPort, r))
}

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS)
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins - change in production!
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
