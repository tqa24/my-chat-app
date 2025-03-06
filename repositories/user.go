package repositories

import (
	"errors"
	"log"
	"my-chat-app/models"
	"time"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByUsername(username string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByID(id string) (*models.User, error)
	GetAll() ([]models.User, error)
	Update(user *models.User) error
	GetUnverifiedWithExpiredOTP() ([]*models.User, error)
	SoftDelete(user *models.User) error
	GetByUsernameIncludingDeleted(username string) (*models.User, error)
	GetByEmailIncludingDeleted(email string) (*models.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	return &user, err
}
func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *userRepository) GetByID(id string) (*models.User, error) {
	var user models.User
	log.Printf("GetByID: UserID: %s", id)
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
	return &user, err
}
func (r *userRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}
func (r *userRepository) GetAll() ([]models.User, error) {
	var users []models.User
	err := r.db.Where("deleted_at IS NULL AND is_verified = true").Find(&users).Error
	return users, err
}

// GetUnverifiedWithExpiredOTP fetches unverified accounts with expired OTPs
func (r *userRepository) GetUnverifiedWithExpiredOTP() ([]*models.User, error) {
	var users []*models.User

	// Find users that:
	// 1. Are not verified
	// 2. Have an OTP expiry time that has passed
	// 3. Are not already soft deleted
	err := r.db.Where("is_verified = ? AND otp_expiry < ? AND deleted_at IS NULL",
		false, time.Now()).Find(&users).Error

	return users, err
}

// SoftDelete marks a user as deleted without removing from the database
func (r *userRepository) SoftDelete(user *models.User) error {
	now := time.Now()
	user.DeletedAt = &now
	return r.db.Save(user).Error
}
func (r *userRepository) GetByUsernameIncludingDeleted(username string) (*models.User, error) {
	var user models.User
	err := r.db.Unscoped().Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmailIncludingDeleted(email string) (*models.User, error) {
	var user models.User
	err := r.db.Unscoped().Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
}
