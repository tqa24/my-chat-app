package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm" // Import gorm
)

type Message struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SenderID   uuid.UUID `gorm:"type:uuid;not null" json:"sender_id"`
	ReceiverID uuid.UUID `gorm:"type:uuid;not null" json:"receiver_id"` // For direct messages
	Content    string    `gorm:"not null" json:"content"`
	Status     string    `gorm:"default:sent" json:"status"` // sent, received, read
	CreatedAt  time.Time `json:"created_at"`
	Sender     User      `gorm:"foreignKey:SenderID;references:ID" json:"-"`   // Don't include in JSON
	Receiver   User      `gorm:"foreignKey:ReceiverID;references:ID" json:"-"` // Don't include in JSON
}

// BeforeCreate hook to generate UUID for the message ID.
func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	m.ID = uuid.New() // Generate a new UUID
	return
}
