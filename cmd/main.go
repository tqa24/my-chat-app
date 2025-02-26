package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"my-chat-app/api"
	"my-chat-app/config"
	"my-chat-app/consumer"
	"my-chat-app/repositories"
	"my-chat-app/services"
	"my-chat-app/websockets"

	_ "expvar"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/streadway/amqp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Define Prometheus metrics
var (
	messagesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_app_messages_sent_total",
		Help: "The total number of messages sent",
	})

	messagesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_app_messages_received_total",
		Help: "The total number of messages received",
	})

	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_app_http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"}, // Labels for method, path, and status code
	)

	httpRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "chat_app_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets, // Use default buckets (or customize)
		},
		[]string{"method", "path"}, // Labels for method and path
	)

	activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_app_active_connections",
		Help: "Number of active WebSocket connections.",
	})

	loginAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_app_login_attempts_total",
		Help: "Total number of login attempts.",
	})

	failedLoginAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_app_failed_login_attempts_total",
		Help: "Total number of failed login attempts.",
	})

	messagesInQueue = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_app_messages_in_queue",
		Help: "Number of messages currently in the RabbitMQ queue.",
	})

	databaseQueryDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "chat_app_database_query_duration_seconds",
			Help:    "Duration of database queries in seconds.",
			Buckets: prometheus.DefBuckets, // Or customize buckets
		},
		[]string{"query"}, // Label for the type of query (e.g., "get_conversation", "create_message")
	)
)

// Gin middleware for HTTP request metrics
func prometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath() // Use FullPath to get the route pattern (e.g., "/messages/:id")
		method := c.Request.Method

		c.Next() // Process the request

		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", c.Writer.Status())

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDurationSeconds.WithLabelValues(method, path).Observe(duration)
	}
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Load configuration
	config.LoadConfig()
	// Connect to PostgreSQL
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.AppConfig.DBHost, config.AppConfig.DBUser, config.AppConfig.DBPassword, config.AppConfig.DBName, config.AppConfig.DBPort)

	// Wrap DB connection for monitoring
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	// Wrap db
	wrappedDB := &wrappedGormDB{db}

	// Connect to RabbitMQ
	conn, err := amqp.Dial(config.AppConfig.RabbitMQURL)
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Failed to open a channel:", err)
	}
	defer ch.Close()

	// Declare the queue
	_, err = ch.QueueDeclare(
		"chat_queue", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		log.Fatal("Failed to declare a queue:", err)
	}

	// Initialize repositories. Use wrappedDB.DB
	userRepo := repositories.NewUserRepository(wrappedDB.DB)       // Pass wrappedDB.DB
	messageRepo := repositories.NewMessageRepository(wrappedDB.DB) // Pass wrappedDB.DB
	groupRepo := repositories.NewGroupRepository(wrappedDB.DB)     // Pass wrappedDB.DB

	// Initialize WebSocket hub
	hub := websockets.NewHub()
	go hub.Run() // Run the hub in a separate goroutine

	// Monitor active connections
	go func() {
		for {
			activeConnections.Set(float64(len(hub.Clients)))
			time.Sleep(5 * time.Second) // Update every 5 seconds (adjust as needed)
		}
	}()

	// Initialize AI service
	aiService, err := services.NewAIService()
	if err != nil {
		log.Printf("Warning: Failed to initialize AI service: %v", err)
	}

	// Initialize services
	authService := services.NewAuthService(userRepo)
	chatService := services.NewChatService(messageRepo, groupRepo, userRepo, hub, aiService) // Inject the hub
	groupService := services.NewGroupService(groupRepo, userRepo, hub)

	// Initialize handlers
	authHandler := api.NewAuthHandler(authService, userRepo)
	chatHandler := api.NewChatHandler(chatService, hub, wrappedDB.DB, ch) // Use wrappedDB.DB and Pass the amqp channel
	groupHandler := api.NewGroupHandler(groupService)

	// Expose Prometheus metrics
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":6060", nil))
	}()

	// Initialize Gin router
	r := gin.Default()

	// Use the Prometheus middleware
	r.Use(prometheusMiddleware())

	//CORS
	r.Use(CORSMiddleware())

	// *** IMPORTANT: Define API routes *BEFORE* serving static files ***
	apiRoutes := r.Group("/api") // Group your API routes under /api
	{
		apiRoutes.POST("/register", authHandler.Register)
		// Wrap login
		apiRoutes.POST("/login", func(c *gin.Context) {
			loginAttempts.Inc()
			authHandler.Login(c)
		})

		apiRoutes.POST("/logout", authHandler.Logout)
		apiRoutes.GET("/profile", authHandler.Profile)
		apiRoutes.GET("/ws", chatHandler.WebSocketHandler)
		apiRoutes.GET("/messages", chatHandler.GetConversation) // For get conversation
		apiRoutes.POST("/messages", func(c *gin.Context) {
			messagesSent.Inc()
			chatHandler.SendMessage(c)
		})
		apiRoutes.GET("/users", authHandler.GetAllUsers)
		// Group routes
		apiRoutes.POST("/groups", groupHandler.CreateGroup)                     // Create a new group
		apiRoutes.GET("/groups/:id", groupHandler.GetGroup)                     // Get group details
		apiRoutes.POST("/groups/:id/join", groupHandler.JoinGroup)              // Join a group
		apiRoutes.POST("/groups/join-by-code", groupHandler.JoinGroupByCode)    // Join group by code
		apiRoutes.POST("/groups/:id/leave", groupHandler.LeaveGroup)            // Leave a group
		apiRoutes.GET("/users/:id/groups", groupHandler.ListGroupsForUser)      // List groups for a user
		apiRoutes.GET("/groups", groupHandler.GetAllGroups)                     // Get all groups
		apiRoutes.GET("/groups/:id/messages", chatHandler.GetGroupConversation) // For get group conversation
		apiRoutes.POST("/messages/:id/react", chatHandler.AddReaction)          // Add reaction
		apiRoutes.DELETE("/messages/:id/react", chatHandler.RemoveReaction)     // Remove reaction
		apiRoutes.GET("/groups/:id/members", groupHandler.GetGroupMembers)      // Get member from group
		// *** File Upload Route ***
		apiRoutes.POST("/upload", chatHandler.UploadFile)

	}

	// Serve static files from 'frontend/dist', but under a /static prefix
	r.Static("/static", "./frontend/dist")

	// *** Serve uploaded files statically ***
	r.Static("/uploads", "./uploads")

	// IMPORTANT:  Handle the index.html fallback explicitly.
	// This serves index.html for any route that isn't /api/* or /static/* or /uploads/*
	r.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api") &&
			!strings.HasPrefix(c.Request.URL.Path, "/static") &&
			!strings.HasPrefix(c.Request.URL.Path, "/uploads") {
			c.File("./frontend/dist/index.html")
		} else {
			c.AbortWithStatus(http.StatusNotFound) // Or a custom 404 page
		}
	})

	// ... (rest of your main function - NO CHANGES HERE) ...
	// --- START CONSUMER ---  VERY IMPORTANT!
	go func() {
		consumerService, err := consumer.NewConsumer(config.AppConfig.RabbitMQURL, chatService)
		if err != nil {
			log.Fatalf("Failed to create consumer: %v", err)
		}

		// Wrap the consumer's message processing to increment a counter
		wrappedChatService := &wrappedChatService{
			ChatService: consumerService.ChatService,
		}
		consumerService.ChatService = wrappedChatService

		// Monitor messages in queue
		go func() {
			for {
				q, err := ch.QueueInspect("chat_queue") // Use QueueInspect to get queue stats
				if err != nil {
					log.Printf("Failed to inspect queue: %v", err)
				} else {
					messagesInQueue.Set(float64(q.Messages)) // Get message from Queue
				}
				time.Sleep(5 * time.Second)
			}
		}()

		if err := consumerService.StartConsuming(); err != nil {
			log.Fatalf("Consumer error: %v", err)
		}

	}()

	// Start the server
	log.Printf("Server listening on port %s", config.AppConfig.AppPort)
	log.Fatal(http.ListenAndServe(":"+config.AppConfig.AppPort, r))
}

// ... (rest of your helper functions - NO CHANGES HERE) ...
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

// Helper function to wrap services.ChatService for counting received message.
type wrappedChatService struct {
	services.ChatService
}

func (w *wrappedChatService) SendMessage(senderID, receiverID, groupID, content, replyToMessageID, fileName, filePath, fileType string, fileSize int64, checksum string) (string, error) {
	messagesReceived.Inc() // Increment received message.
	return w.ChatService.SendMessage(senderID, receiverID, groupID, content, replyToMessageID, fileName, filePath, fileType, fileSize, checksum)
}

// Helper function to wrap gorm.DB for counting database query.
type wrappedGormDB struct {
	*gorm.DB
}

// Wrap all db method
func (db *wrappedGormDB) Create(value interface{}) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.Create(value)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("create").Observe(duration)
	return
}

func (db *wrappedGormDB) Where(query interface{}, args ...interface{}) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.Where(query, args...)
	duration := time.Since(start).Seconds()
	//You can adjust this to identify kind of where query by add more condition.
	databaseQueryDurationSeconds.WithLabelValues("where").Observe(duration)
	return
}

func (db *wrappedGormDB) Preload(query string, args ...interface{}) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.Preload(query, args...)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("preload").Observe(duration)
	return
}

func (db *wrappedGormDB) First(dest interface{}, conds ...interface{}) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.First(dest, conds...)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("first").Observe(duration) // General "first" query
	return
}

func (db *wrappedGormDB) Find(dest interface{}, conds ...interface{}) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.Find(dest, conds...)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("find").Observe(duration)
	return
}

func (db *wrappedGormDB) Save(value interface{}) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.Save(value)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("save").Observe(duration)
	return
}

func (db *wrappedGormDB) Delete(value interface{}, conds ...interface{}) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.Delete(value, conds...)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("delete").Observe(duration)
	return
}

func (db *wrappedGormDB) Raw(sql string, values ...interface{}) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.Raw(sql, values...)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("raw").Observe(duration)
	return
}

func (db *wrappedGormDB) Exec(sql string, values ...interface{}) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.Exec(sql, values...)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("exec").Observe(duration)
	return
}

func (db *wrappedGormDB) Model(value interface{}) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.Model(value)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("model").Observe(duration)
	return
}

func (db *wrappedGormDB) Association(column string) *gorm.Association {
	start := time.Now()
	association := db.DB.Association(column)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("association").Observe(duration)
	return association
}

func (db *wrappedGormDB) Count(count *int64) (tx *gorm.DB) {
	start := time.Now()
	tx = db.DB.Count(count)
	duration := time.Since(start).Seconds()
	databaseQueryDurationSeconds.WithLabelValues("count").Observe(duration)
	return
}
