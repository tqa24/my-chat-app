package services

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"my-chat-app/config"
	"my-chat-app/models"
	"my-chat-app/repositories"
	"strings"
	"time"

	"github.com/go-mail/mail/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	RegisterUser(user *models.User) error
	LoginUser(username, password string) (*models.User, error)
	LoginUserWithEmail(email, password string) (*models.User, error)
	GetUserProfile(userID string) (*models.User, error)
	VerifyOTP(email, otp string) error
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

	// Check for disposable email.
	if isDisposableEmail(user.Email) {
		return errors.New("disposable email addresses are not allowed")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	log.Printf("Register info after hash, %+v", user) //Moved

	log.Printf("Register info after hash, %+v", user)

	// Generate OTP and set expiry.
	otp := generateOTP()
	otpExpiry := time.Now().Add(10 * time.Minute) // OTP expires in 10 minutes.
	user.OTP = otp
	user.OTPExpiry = &otpExpiry

	if err := s.sendOTPEmail(user.Email, otp); err != nil {
		//  Don't save the user if the email fails.
		log.Printf("Error sending OTP email: %v", err)
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	// Create the user (but they are not fully registered until OTP is verified).
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

// isDisposableEmail checks if the given email is from a disposable email provider.
func isDisposableEmail(email string) bool {
	// Very Basic List for example.  You'll need a more robust solution.
	disposableDomains := []string{
		"mailinator.com",
		"guerrillamail.com",
		"tempmail.com",
		// ... add many more ...
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false // Invalid email format
	}
	domain := parts[1]

	for _, disposableDomain := range disposableDomains {
		if domain == disposableDomain {
			return true
		}
	}
	return false
}

// generateOTP generates a 6-digit OTP.
func generateOTP() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func (s *authService) sendOTPEmail(email, otp string) error {
	m := mail.NewMessage()
	m.SetHeader("From", config.AppConfig.EmailFrom)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Your OTP for Chat App Registration")
	m.SetBody("text/plain", fmt.Sprintf("Your OTP is: %s", otp))

	d := mail.NewDialer(config.AppConfig.EmailHost, config.AppConfig.EmailPort, config.AppConfig.EmailUsername, config.AppConfig.EmailPassword)

	d.StartTLSPolicy = mail.MandatoryStartTLS

	// Attempt to send the email.
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

// VerifyOTP verifies the provided OTP against the stored OTP for the user.
func (s *authService) VerifyOTP(email, otp string) error {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Check if OTP is expired
	if user.OTPExpiry == nil || time.Now().After(*user.OTPExpiry) {
		return errors.New("OTP has expired")
	}

	if user.OTP != otp {
		return errors.New("invalid OTP")
	}

	// Clear the OTP after successful verification.
	user.OTP = ""
	user.OTPExpiry = nil
	return s.userRepo.Update(user) // Save the changes to the user.
}
