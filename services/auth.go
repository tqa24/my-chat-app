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
	//Change below
	LoginUser(username, password string) (*models.User, error)
	LoginUserWithEmail(email, password string) (*models.User, error)
	GetUserProfile(userID string) (*models.User, error)
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
	log.Printf("Register info before hash, %+v", user)
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
	log.Printf("Register info after hash, %+v", user) //Moved

	// Create the user
	return s.userRepo.Create(user)
}

func (s *authService) LoginUser(username, password string) (*models.User, error) {
	log.Printf("Login with username %v %v", username, password)
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		log.Printf("LoginUser: User not found by username: %s, error: %v", username, err) // Log user not found
		return nil, errors.New("invalid credentials")
	}

	log.Printf("LoginUser: User found: %+v", user)                      // Log the user object
	log.Printf("LoginUser: Hashed password from DB: %s", user.Password) // Log hashed password

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	log.Printf("LoginUser: bcrypt.CompareHashAndPassword result: %v", err) // Log bcrypt result

	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

func (s *authService) LoginUserWithEmail(email, password string) (*models.User, error) {
	log.Printf("Login with email %v %v", email, password)
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		log.Printf("LoginUserWithEmail: User not found by email: %s, error: %v", email, err) // Log user not found
		return nil, errors.New("invalid credentials")
	}

	log.Printf("LoginUserWithEmail: User found: %+v", user)                      // Log the user object
	log.Printf("LoginUserWithEmail: Hashed password from DB: %s", user.Password) // Log hashed password

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	log.Printf("LoginUserWithEmail: bcrypt.CompareHashAndPassword result: %v", err) // Log bcrypt result

	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

func (s *authService) GetUserProfile(userID string) (*models.User, error) {
	return s.userRepo.GetByID(userID)
}
