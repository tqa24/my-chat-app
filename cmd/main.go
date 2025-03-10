package main

import (
	"fmt"
	"log"
	"my-chat-app/middleware"
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

	messageRetryCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_app_message_retry_count",
			Help: "Number of message retry attempts.",
		},
		[]string{"retry_number"}, // Label for retry attempt number
	)

	deadLetterMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_app_dead_letter_messages_total",
		Help: "Total number of messages sent to dead letter queue.",
	})
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
	jwtService := services.NewJWTService()
	authService := services.NewAuthService(userRepo, jwtService)
	chatService := services.NewChatService(messageRepo, groupRepo, userRepo, hub, aiService) // Inject the hub
	groupService := services.NewGroupService(groupRepo, userRepo, hub)

	// Initialize and start the cleanup service
	cleanupService := services.NewCleanupService(userRepo)
	cleanupService.StartCleanupScheduler(24 * time.Hour) // Run cleanup once a day

	// Initialize handlers
	authHandler := api.NewAuthHandler(authService, userRepo)
	chatHandler := api.NewChatHandler(chatService, hub, wrappedDB.DB, ch, jwtService) // Use wrappedDB.DB and Pass the amqp channel
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

	// Use the message size limiter middleware
	r.Use(middleware.MessageSizeLimiter())

	//CORS
	r.Use(CORSMiddleware())

	// *** IMPORTANT: Define API routes *BEFORE* serving static files ***
	apiRoutes := r.Group("/api") // Group your API routes under /api

	// Public routes (no JWT required)
	apiRoutes.POST("/register", authHandler.Register)
	apiRoutes.POST("/login", func(c *gin.Context) {
		loginAttempts.Inc()
		authHandler.Login(c)
	})
	apiRoutes.POST("/verify-otp", authHandler.VerifyOTP)
	apiRoutes.POST("/resend-otp", authHandler.ResendOTP)
	apiRoutes.POST("/logout", authHandler.Logout) // Often logout is public as it just clears client-side tokens

	// Protected routes (JWT required)
	protected := apiRoutes.Group("/")
	protected.Use(middleware.JWTAuthMiddleware(jwtService))
	{
		// User routes
		protected.GET("/profile", authHandler.Profile)
		protected.GET("/users", authHandler.GetAllUsers)

		// WebSocket route
		protected.GET("/ws", chatHandler.WebSocketHandler)

		// Message routes
		protected.GET("/messages", chatHandler.GetConversation)
		protected.POST("/messages", func(c *gin.Context) {
			messagesSent.Inc()
			chatHandler.SendMessage(c)
		})
		protected.POST("/messages/:id/react", chatHandler.AddReaction)
		protected.DELETE("/messages/:id/react", chatHandler.RemoveReaction)

		// Group routes
		protected.POST("/groups", groupHandler.CreateGroup)
		protected.GET("/groups/:id", groupHandler.GetGroup)
		protected.POST("/groups/:id/join", groupHandler.JoinGroup)
		protected.POST("/groups/join-by-code", groupHandler.JoinGroupByCode)
		protected.POST("/groups/:id/leave", groupHandler.LeaveGroup)
		protected.GET("/users/:id/groups", groupHandler.ListGroupsForUser)
		protected.GET("/groups", groupHandler.GetAllGroups)
		protected.GET("/groups/:id/messages", chatHandler.GetGroupConversation)
		protected.GET("/groups/:id/members", groupHandler.GetGroupMembers)

		// File upload route
		protected.POST("/upload", chatHandler.UploadFile)
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

	// --- START CONSUMER
	go func() {
		consumerService, err := consumer.NewConsumer(
			config.AppConfig.RabbitMQURL,
			chatService,
			messageRetryCount,
			deadLetterMessages,
		)
		if err != nil {
			log.Fatalf("Failed to create consumer: %v", err)
		}

		// Wrap the consumer's message processing to increment a counter
		wrappedChatService := &wrappedChatService{
			ChatService: consumerService.ChatService,
		}
		consumerService.ChatService = wrappedChatService

		// Start processing the dead letter queue
		if err := consumerService.ProcessDeadLetterQueue(); err != nil {
			log.Printf("Failed to start DLQ consumer: %v", err)
		}

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
