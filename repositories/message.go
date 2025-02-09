package repositories

import (
	"my-chat-app/models"

	"gorm.io/gorm"
)

type MessageRepository interface {
	Create(message *models.Message) error
	GetConversation(user1ID, user2ID string, limit, offset int) ([]models.Message, error)
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db}
}

func (r *messageRepository) Create(message *models.Message) error {
	return r.db.Create(message).Error
}

func (r *messageRepository) GetConversation(user1ID, user2ID string, limit, offset int) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.
		Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)", user1ID, user2ID, user2ID, user1ID).
		Order("created_at desc"). // Most recent first
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}
