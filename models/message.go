package models

import (
	"time"

	"github.com/google/uuid"
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
