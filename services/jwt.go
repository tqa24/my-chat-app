package services

import (
	"errors"
	"fmt"
	"my-chat-app/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService interface {
	GenerateToken(userID uuid.UUID, username string) (string, error)
	ValidateToken(tokenString string) (*jwt.Token, error)
	GetUserIDFromToken(token *jwt.Token) (string, error)
}

type jwtService struct {
	secretKey string
	issuer    string
}

// JWTCustomClaims contains custom data we want in the token
type JWTCustomClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewJWTService() JWTService {
	return &jwtService{
		secretKey: config.AppConfig.JWTSecret,
		issuer:    "chat-app",
	}
}

func (s *jwtService) GenerateToken(userID uuid.UUID, username string) (string, error) {
	// Set expiration time (e.g., 24 hours from now)
	expirationTime := time.Now().Add(24 * time.Hour)

	// Create the JWT claims
	claims := &JWTCustomClaims{
		UserID:   userID.String(),
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.issuer,
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate the signed token
	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *jwtService) ValidateToken(tokenString string) (*jwt.Token, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&JWTCustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(s.secretKey), nil
		},
	)

	if err != nil {
		return nil, err
	}

	// Verify token is valid
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return token, nil
}

func (s *jwtService) GetUserIDFromToken(token *jwt.Token) (string, error) {
	claims, ok := token.Claims.(*JWTCustomClaims)
	if !ok {
		return "", errors.New("failed to parse claims")
	}
	return claims.UserID, nil
}
