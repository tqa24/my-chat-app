package api

import (
	"github.com/google/uuid"
	"log" // Import log
	"my-chat-app/repositories"
	"net/http"
	"strings"

	"my-chat-app/models"
	"my-chat-app/services"
	"my-chat-app/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService services.AuthService
	userRepo    repositories.UserRepository // Inject UserRepository
}

func NewAuthHandler(authService services.AuthService, userRepo repositories.UserRepository) *AuthHandler {
	return &AuthHandler{authService, userRepo}
}

func (h *AuthHandler) Register(c *gin.Context) {
	log.Println("Register handler called") // Log entry point
	var user models.User                   //This line
	if err := c.ShouldBindJSON(&user); err != nil {
		log.Printf("Register: Error binding JSON: %v", err) // Log binding errors
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}
	log.Printf("Register: Received user data: %+v", user) // Log received data

	err := h.authService.RegisterUser(&user)
	if err != nil {
		log.Printf("Register: Error from authService: %v", err) // Log service errors
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	log.Println("Register: User registered successfully") // Log success
	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	log.Println("Login handler called") // Log entry point
	var credentials struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		log.Printf("Login: Error binding JSON: %v", err) // Log binding errors
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}
	log.Printf("Login: Received credentials: %+v", credentials) // Log received data

	// Trim whitespace from identifier and password
	credentials.Identifier = strings.TrimSpace(credentials.Identifier)
	credentials.Password = strings.TrimSpace(credentials.Password)

	log.Printf("Login: Trimmed credentials: %+v", credentials) // Log trimmed data

	var user *models.User
	var err error

	// Check if the identifier is an email
	if strings.Contains(credentials.Identifier, "@") {
		log.Println("Login: Attempting login with email")
		user, err = h.authService.LoginUserWithEmail(credentials.Identifier, credentials.Password)
	} else {
		log.Println("Login: Attempting login with username")
		user, err = h.authService.LoginUser(credentials.Identifier, credentials.Password)
	}

	if err != nil {
		log.Printf("Login: Error from authService: %v", err) // Log service errors
		utils.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	log.Println("Login: Login successful") // Log success
	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "user": user})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// In a real application, you would invalidate the user's session or JWT here.
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}
func (h *AuthHandler) Profile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get Profile"})
}
func (h *AuthHandler) GetAllUsers(c *gin.Context) {
	users, err := h.userRepo.GetAll()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}
	type UserResponse struct {
		ID       uuid.UUID `json:"id"`
		Username string    `json:"username"`
		Email    string    `json:"email"`
	}
	var userResponses []UserResponse
	for _, user := range users {
		userResponses = append(userResponses, UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		})
	}
	c.JSON(http.StatusOK, userResponses)
}
