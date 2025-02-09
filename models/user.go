package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username  string    `gorm:"unique;not null" json:"username"`
	Password  string    `gorm:"not null" json:"password"`
	Email     string    `gorm:"unique;not null" json:"email"`
	LastSeen  time.Time `gorm:"type:timestamp with time zone" json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate hook to generate UUID for ID
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = uuid.New()
	return
}
