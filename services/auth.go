package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"my-chat-app/config"
	"my-chat-app/models"
	"my-chat-app/repositories"
	"net/http"
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

type mailcheckResponse struct {
	Valid      bool   `json:"valid"`
	Block      bool   `json:"block"`
	Disposable bool   `json:"disposable"`
	Domain     string `json:"domain"`
	Text       string `json:"text"`
	Reason     string `json:"reason"`
	Risk       int    `json:"risk"`
}

func (s *authService) RegisterUser(user *models.User) error {
	// Trim whitespace
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.TrimSpace(user.Email)
	user.Password = strings.TrimSpace(user.Password)
	user.IsVerified = false

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

	// Use the Mailcheck API to check for disposable email.
	isDisposable, err := s.isDisposableEmail(user.Email)
	if err != nil {
		log.Printf("Error checking disposable email: %v", err)
		//  Handle the error appropriately.  You might choose *not* to block
		// registration if the API is unavailable, or you might.
		return fmt.Errorf("failed to check disposable email: %w", err)
	}
	if isDisposable {
		return errors.New("disposable email addresses are not allowed")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	log.Printf("Register info after hash, %+v", user)

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
	// Check if the user is verified
	if !user.IsVerified {
		return nil, errors.New("Account not verified. Please check your email for the OTP")
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
	// Check if the user is verified
	if !user.IsVerified {
		return nil, errors.New("Account not verified. Please check your email for the OTP")
	}
	return user, nil
}

func (s *authService) GetUserProfile(userID string) (*models.User, error) {
	return s.userRepo.GetByID(userID)
}

// isDisposableEmail checks if the given email is from a disposable email provider.
func (s *authService) isDisposableEmail(email string) (bool, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid email format")
	}
	domain := parts[1]

	url := fmt.Sprintf("https://mailcheck.p.rapidapi.com/?domain=%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("x-rapidapi-key", config.AppConfig.RapidAPIKey) // Use the key from config
	req.Header.Add("x-rapidapi-host", "mailcheck.p.rapidapi.com")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make API request: %w", err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("mailcheck API returned non-200 status: %d, body: %s", res.StatusCode, string(body))
	}

	var result mailcheckResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	log.Printf("Mail Check result %+v", result)
	return result.Block || result.Disposable, nil
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

	//d.TLSConfig = &tls.Config{
	//	ServerName:         config.AppConfig.EmailHost, // Set ServerName for TLS verification
	//	InsecureSkipVerify: false,                   // MUST be false in production.
	//	// You might need MinVersion: tls.VersionTLS12 if the server doesn't support TLS 1.3
	//}

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
	user.IsVerified = true         // Set is_verified to true after successful OTP verification
	return s.userRepo.Update(user) // Save the changes to the user.
}
