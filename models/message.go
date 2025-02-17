package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Message struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SenderID         uuid.UUID      `gorm:"type:uuid;not null" json:"sender_id"`
	ReceiverID       *uuid.UUID     `gorm:"type:uuid" json:"receiver_id"` //Nullable for group chat
	GroupID          *uuid.UUID     `gorm:"type:uuid" json:"group_id"`    // Add GroupID, nullable
	Content          string         `gorm:"not null" json:"content"`
	Status           string         `gorm:"default:sent" json:"status"` // sent, received, read
	CreatedAt        time.Time      `json:"created_at"`
	Sender           *User          `gorm:"foreignKey:SenderID;references:ID" json:"sender"`                             // Don't include in JSON
	Receiver         *User          `gorm:"foreignKey:ReceiverID;references:ID" json:"receiver"`                         // Don't include in JSON
	Group            *Group         `gorm:"foreignKey:GroupID;references:ID" json:"group"`                               // Add Group
	Reactions        datatypes.JSON `gorm:"type:jsonb" json:"reactions"`                                                 // NEW: Reactions as JSONB
	ReplyToMessageID *uuid.UUID     `gorm:"type:uuid" json:"reply_to_message_id"`                                        // NEW: Reply-to ID
	ReplyToMessage   *Message       `gorm:"foreignKey:ReplyToMessageID;references:ID" json:"reply_to_message,omitempty"` // NEW: Include the replied-to message (optional)
	// *** NEW: File Upload Fields ***
	FileName     string `gorm:"type:varchar(255)" json:"file_name"` // Original filename
	FilePath     string `gorm:"type:varchar(255)" json:"file_path"` // Path to stored file (relative to upload dir)
	FileType     string `gorm:"type:varchar(100)" json:"file_type"` // image/jpeg, application/pdf, etc.
	FileSize     int64  `gorm:"type:bigint" json:"file_size"`       // File size in bytes
	FileChecksum string `gorm:"type:varchar(64)" json:"checksum"`   // Add the checksum field
}

// BeforeCreate hook to generate UUID for the message ID.
func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New() // Generate a new UUID
	}
	// Initialize Reactions to an empty JSON object if it's nil
	if m.Reactions == nil {
		m.Reactions = datatypes.JSON([]byte("{}"))
	}
	return
}
