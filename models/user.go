package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username           string     `gorm:"unique;not null" json:"username"`
	Password           string     `gorm:"not null" json:"password"`
	Email              string     `gorm:"unique;not null" json:"email"`
	LastSeen           time.Time  `gorm:"type:timestamp with time zone" json:"last_seen"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `gorm:"type:timestamp with time zone" json:"-"`
	OTP                string     `gorm:"type:varchar(6)" json:"-"`
	OTPExpiry          *time.Time `gorm:"type:timestamp with time zone" json:"-"`
	IsVerified         bool       `gorm:"default:false" json:"is_verified"`
	OTPAttempts        int        `gorm:"default:0" json:"-"`
	OTPAttemptsResetAt *time.Time `gorm:"type:timestamp with time zone" json:"-"`
	Groups             []*Group   `gorm:"many2many:user_groups;" json:"groups"`
}

// BeforeCreate hook to generate UUID for ID
func (u *User) BeforeCreate(*gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if u.OTPAttemptsResetAt == nil {
		now := time.Now()
		u.OTPAttemptsResetAt = &now
	}
	return
}
