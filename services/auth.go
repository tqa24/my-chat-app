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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const maxOTPRetries = 5
const otpRetryResetDuration = 3 * 24 * time.Hour // 3 days

type AuthService interface {
	RegisterUser(user *models.User) error
	LoginUser(username, password string) (*models.User, string, error)
	LoginUserWithEmail(email, password string) (*models.User, string, error)
	GetUserProfile(userID string) (*models.User, error)
	VerifyOTP(email, otp string) error
	ResendOTP(email string) error
}

type authService struct {
	userRepo   repositories.UserRepository
	jwtService JWTService
}

func NewAuthService(userRepo repositories.UserRepository, jwtService JWTService) AuthService {
	return &authService{userRepo, jwtService}
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

	// Check if username already exists (including soft-deleted accounts)
	existingByUsername, err := s.userRepo.GetByUsernameIncludingDeleted(user.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // Return any error that's NOT ErrRecordNotFound
	}

	// Check if email already exists (including soft-deleted accounts)
	existingByEmail, err := s.userRepo.GetByEmailIncludingDeleted(user.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // Return any error that's NOT ErrRecordNotFound
	}

	// Case 1: Both username and email exist (must be the same account to reactivate)
	if existingByUsername != nil && existingByEmail != nil {
		// If they're different accounts, that's an error
		if existingByUsername.ID != existingByEmail.ID {
			return errors.New("username and email belong to different accounts")
		}

		// They're the same account - check if it's soft-deleted
		if existingByUsername.DeletedAt == nil {
			return errors.New("account already exists and is active")
		}

		// Reactivate the soft-deleted account
		return s.reactivateAccount(existingByUsername, user.Password)
	}

	// Case 2: Only username exists
	if existingByUsername != nil {
		if existingByUsername.DeletedAt == nil {
			return errors.New("username already exists")
		}
		// Username exists but with a different email
		return errors.New("to reactivate your account, please use the same email address previously associated with this username")
	}

	// Case 3: Only email exists
	if existingByEmail != nil {
		if existingByEmail.DeletedAt == nil {
			return errors.New("email already exists")
		}
		// Email exists but with a different username
		return errors.New("to reactivate your account, please use the same username previously associated with this email address")
	}

	// Case 4: Neither exists - proceed with new registration
	return s.createNewAccount(user)
}

// Helper method to reactivate an account
func (s *authService) reactivateAccount(existingUser *models.User, newPassword string) error {
	log.Printf("Reactivating soft-deleted account for user: %s", existingUser.Username)

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update the existing user
	existingUser.Password = string(hashedPassword)
	existingUser.DeletedAt = nil // Reactivate by clearing DeletedAt
	existingUser.IsVerified = false

	// Generate OTP and set expiry for verification
	otp := generateOTP()
	otpExpiry := time.Now().Add(10 * time.Minute)
	existingUser.OTP = otp
	existingUser.OTPExpiry = &otpExpiry

	// Send OTP email
	if err := s.sendOTPEmail(existingUser.Email, otp); err != nil {
		log.Printf("Error sending OTP email: %v", err)
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	// Update the user record
	return s.userRepo.Update(existingUser)
}

// Helper method to create a new account
func (s *authService) createNewAccount(user *models.User) error {
	// Check for disposable email
	isDisposable, err := s.isDisposableEmail(user.Email)
	if err != nil {
		log.Printf("Error checking disposable email: %v", err)
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

	// Generate OTP and set expiry
	otp := generateOTP()
	otpExpiry := time.Now().Add(1 * time.Minute) // OTP expires in 10 minutes.
	user.OTP = otp
	user.OTPExpiry = &otpExpiry

	if err := s.sendOTPEmail(user.Email, otp); err != nil {
		log.Printf("Error sending OTP email: %v", err)
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	// Create the user (but they are not fully registered until OTP is verified).
	return s.userRepo.Create(user)
}

func (s *authService) LoginUser(username, password string) (*models.User, string, error) {
	log.Printf("Login with username %v %v", username, password)
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		log.Printf("LoginUser: User not found by username: %s, error: %v", username, err) // Log user not found
		return nil, "", errors.New("invalid credentials")
	}

	// Check if the account is soft deleted
	if user.DeletedAt != nil {
		log.Printf("LoginUser: Attempt to login to deleted account: %s", username)
		return nil, "", errors.New("account has been deactivated")
	}

	log.Printf("LoginUser: User found: %+v", user)
	log.Printf("LoginUser: Hashed password from DB: %s", user.Password)
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	log.Printf("LoginUser: bcrypt.CompareHashAndPassword result: %v", err) // Log bcrypt result
	if err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	// Check if the user is verified
	if !user.IsVerified {
		// Check if OTP is expired
		if user.OTPExpiry != nil && time.Now().After(*user.OTPExpiry) {
			// Soft delete the account if OTP is expired
			now := time.Now()
			user.DeletedAt = &now
			if err := s.userRepo.Update(user); err != nil {
				log.Printf("Error soft deleting expired account: %v", err)
			}
			return nil, "", errors.New("verification period has expired, please register again")
		}
		return nil, "", errors.New("account not verified. Please check your email for the OTP")
	}
	// Generate JWT token
	token, err := s.jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, nil
}
func (s *authService) LoginUserWithEmail(email, password string) (*models.User, string, error) {
	log.Printf("Login with email %v %v", email, password)
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		log.Printf("LoginUserWithEmail: User not found by email: %s, error: %v", email, err) // Log user not found
		return nil, "", errors.New("invalid credentials")
	}

	// Check if the account is soft deleted
	if user.DeletedAt != nil {
		log.Printf("LoginUserWithEmail: Attempt to login to deleted account: %s", email)
		return nil, "", errors.New("account has been deactivated")
	}

	log.Printf("LoginUserWithEmail: User found: %+v", user)
	log.Printf("LoginUserWithEmail: Hashed password from DB: %s", user.Password)
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	log.Printf("LoginUserWithEmail: bcrypt.CompareHashAndPassword result: %v", err) // Log bcrypt result
	if err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	// Check if the user is verified
	if !user.IsVerified {
		// Check if OTP is expired
		if user.OTPExpiry != nil && time.Now().After(*user.OTPExpiry) {
			// Soft delete the account if OTP is expired
			now := time.Now()
			user.DeletedAt = &now
			if err := s.userRepo.Update(user); err != nil {
				log.Printf("Error soft deleting expired account: %v", err)
			}
			return nil, "", errors.New("verification period has expired, please register again")
		}
		return nil, "", errors.New("account not verified. Please check your email for the OTP")
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}
	return user, token, nil
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
	return result.Block || result.Disposable || result.Text == "Suspicious", nil
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

	// Check if the account is soft deleted
	if user.DeletedAt != nil {
		return errors.New("this account has been deactivated, please register again")
	}

	// Account Expiry Check
	if !user.IsVerified && user.OTPExpiry != nil && time.Now().After(*user.OTPExpiry) {
		// Soft Delete
		now := time.Now()
		user.DeletedAt = &now
		if err := s.userRepo.Update(user); err != nil {
			return fmt.Errorf("failed to soft delete user: %w", err)
		}
		return errors.New("OTP has expired, please register again")
	}

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

// ResendOTP generates and sends a new OTP to the user's email.
func (s *authService) ResendOTP(email string) error {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Check if the account is soft deleted
	if user.DeletedAt != nil {
		return errors.New("this account has been deactivated, please register again")
	}

	// Check if the user is *already* verified.  If so, don't resend.
	if user.IsVerified {
		return errors.New("account is already verified")
	}

	// --- Rate Limiting Logic ---
	now := time.Now()
	if user.OTPAttemptsResetAt != nil && now.After(*user.OTPAttemptsResetAt) {
		// Reset attempts if the reset time has passed.
		user.OTPAttempts = 0
		resetTime := now.Add(otpRetryResetDuration)
		user.OTPAttemptsResetAt = &resetTime
	}

	if user.OTPAttempts >= maxOTPRetries {
		return errors.New("maximum OTP attempts reached. Please try again later")
	}

	user.OTPAttempts++
	// --- End Rate Limiting Logic ---

	otp := generateOTP()
	otpExpiry := now.Add(10 * time.Minute)
	user.OTP = otp
	user.OTPExpiry = &otpExpiry

	if err := s.sendOTPEmail(user.Email, otp); err != nil {
		log.Printf("Error sending OTP email: %v", err)
		user.OTPAttempts--
		return fmt.Errorf("failed to send OTP email: %w", err)
	}
	// Save the updated user (with new OTP and expiry).
	return s.userRepo.Update(user)
}
