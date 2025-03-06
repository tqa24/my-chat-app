package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBPort        string
	AppPort       string
	RabbitMQURL   string
	EmailHost     string
	EmailPort     int
	EmailUsername string
	EmailPassword string
	EmailFrom     string
	JWTSecret     string
	RapidAPIKey   string
}

var AppConfig Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, using environment variables")
	}

	emailPort, _ := strconv.Atoi(getEnv("EMAIL_PORT", ""))
	AppConfig = Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", ""),
		DBName:        getEnv("DB_NAME", ""),
		DBPort:        getEnv("DB_PORT", "5432"),
		AppPort:       getEnv("APP_PORT", "8080"),
		RabbitMQURL:   getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		EmailHost:     getEnv("EMAIL_HOST", ""),
		EmailPort:     emailPort,
		EmailUsername: getEnv("EMAIL_USERNAME", ""),
		EmailPassword: getEnv("EMAIL_PASSWORD", ""),
		EmailFrom:     getEnv("EMAIL_FROM", ""),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		RapidAPIKey:   getEnv("RAPIDAPI_KEY", ""),
	}
	//if AppConfig.RapidAPIKey == "" {
	//	log.Fatal("RAPIDAPI_KEY environment variable must be set")
	//}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
