package services

import (
	"errors"
	"log"
	"my-chat-app/models"
	"my-chat-app/repositories"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	RegisterUser(user *models.User) error
	LoginUser(username, password string) (*models.User, error)
	LoginUserWithEmail(email, password string) (*models.User, error)
}

type authService struct {
	userRepo repositories.UserRepository
}

func NewAuthService(userRepo repositories.UserRepository) AuthService {
	return &authService{userRepo}
}

func (s *authService) RegisterUser(user *models.User) error {
	// Trim whitespace
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.TrimSpace(user.Email)
	user.Password = strings.TrimSpace(user.Password)

	// IMPORTANT: Log the *plain text* password *before* hashing, for debugging.
	log.Printf("Register: Plain text password BEFORE hashing: %s", user.Password)

	// Check if username already exists
	existingUser, err := s.userRepo.GetByUsername(user.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // Return any error that's NOT ErrRecordNotFound
	}
	if existingUser != nil && existingUser.ID != uuid.Nil {
		return errors.New("username already exists")
	}

	// Check if email already exists
	existingEmail, err := s.userRepo.GetByEmail(user.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // Return any error that's NOT ErrRecordNotFound
	}
	if existingEmail != nil && existingEmail.ID != uuid.Nil {
		return errors.New("email already exists")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	// IMPORTANT: Log the *hashed* password, for debugging.
	log.Printf("Register: Hashed password: %s", user.Password)

	// Create the user
	return s.userRepo.Create(user)
}

func (s *authService) LoginUser(username, password string) (*models.User, error) {
	log.Printf("Login with username %v", username) // Log just the username
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		log.Printf("LoginUser: User not found by username: %s, error: %v", username, err) // Log user not found
		return nil, errors.New("invalid credentials")
	}

	// Log the *hashed* password from the database.
	log.Printf("LoginUser: Hashed password from DB: %s", user.Password)

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))

	// Log the *result* of the bcrypt comparison and the provided password.
	log.Printf("LoginUser: bcrypt.CompareHashAndPassword result: %v, Provided Password: %s", err, password)

	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

func (s *authService) LoginUserWithEmail(email, password string) (*models.User, error) {
	log.Printf("Login with email %v", email) // Log just the email
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		log.Printf("LoginUserWithEmail: User not found by email: %s, error: %v", email, err) // Log user not found
		return nil, errors.New("invalid credentials")
	}

	// Log the *hashed* password from the database.
	log.Printf("LoginUserWithEmail: Hashed password from DB: %s", user.Password)

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	// Log the *result* of the bcrypt comparison and the provided password.
	log.Printf("LoginUserWithEmail: bcrypt.CompareHashAndPassword result: %v, Provided Password: %s", err, password)

	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}
