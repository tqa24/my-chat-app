package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBPort      string
	AppPort     string
	RabbitMQURL string
	JWTSecret   string
	GoogleAIKey string
}

var AppConfig Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, using environment variables")
	}

	AppConfig = Config{
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", "@Quocan12"),
		DBName:      getEnv("DB_NAME", "mychatdb"),
		DBPort:      getEnv("DB_PORT", "5432"),
		AppPort:     getEnv("APP_PORT", "8080"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		JWTSecret:   getEnv("JWT_SECRET", "7dbbfaf39a7af292d8e47f6b52df3496d4f63504c097938ccb18f47dc8ac0f54f23a5c5fb279a465d9c1842b62c4fda23dd1c711efce125c0f9aab41f95894b00abfe14db9643955ad486b918e7751bf71bec3f1fc2e6a00afabb31a3e9a38b82de9affff518aa9e4573a3844a8412a4bfcd0f5d14bb4509efc6c6f3273dc36d43ddeb2992f9b5ee35ebda7722912f385a585ab1f779fa72bdfc2f5f2a21da0e8da6ed42565a90cffe6640cd5e835765329d1895c126de0e6486b71807f6cc77eeff60b0c9d74b9d5686f6435bca9128644ade7feeaeedffe1783699478acc2e94ca58b15315215cfba209d9e0ed753c1cf060ef2c3cf9a9b82b46d6db94e785"),
		GoogleAIKey: getEnv("GOOGLE_AI_API_KEY", ""), // NEW
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
