package models

import (
	"gorm.io/gorm"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SenderID   uuid.UUID  `gorm:"type:uuid;not null" json:"sender_id"`
	ReceiverID *uuid.UUID `gorm:"type:uuid" json:"receiver_id"` //Nullable for group chat
	GroupID    *uuid.UUID `gorm:"type:uuid" json:"group_id"`    // Add GroupID, nullable
	Content    string     `gorm:"not null" json:"content"`
	Status     string     `gorm:"default:sent" json:"status"` // sent, received, read
	CreatedAt  time.Time  `json:"created_at"`
	Sender     User       `gorm:"foreignKey:SenderID;references:ID" json:"sender"`     // Don't include in JSON
	Receiver   User       `gorm:"foreignKey:ReceiverID;references:ID" json:"receiver"` // Don't include in JSON
	Group      Group      `gorm:"foreignKey:GroupID;references:ID" json:"group"`       // Add Group
}

//// BeforeCreate hook to generate UUID for the message ID.
//func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
//	m.ID = uuid.New() // Generate a new UUID
//	return
//}

// BeforeCreate hook to generate UUID for the message ID.
func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New() // Generate a new UUID
	}
	return
}
