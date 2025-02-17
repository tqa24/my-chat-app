package models

import (
	"time"

	"github.com/google/uuid"
)

type Group struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"unique;not null"`
	Code      string    `gorm:"unique;not null"`
	Users     []*User   `gorm:"many2many:user_groups;"` // Many-to-many relationship
	CreatedAt time.Time
	UpdatedAt time.Time
}
