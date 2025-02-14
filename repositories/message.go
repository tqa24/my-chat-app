package repositories

import (
	"my-chat-app/models"

	"gorm.io/gorm"
)

type MessageRepository interface {
	Create(message *models.Message) error
	GetConversation(user1ID, user2ID string, limit, offset int) ([]models.Message, int64, error) // Return messages and total count
	GetGroupConversation(groupID string, limit, offset int) ([]models.Message, int64, error)     // Return messages and total count
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

func (r *messageRepository) GetConversation(user1ID, user2ID string, limit, offset int) ([]models.Message, int64, error) {
	var messages []models.Message
	var count int64

	// Get the total count of messages
	r.db.Model(&models.Message{}).
		Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)", user1ID, user2ID, user2ID, user1ID).
		Count(&count)

	// Get the messages with limit and offset
	err := r.db.
		Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)", user1ID, user2ID, user2ID, user1ID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error

	return messages, count, err
}

func (r *messageRepository) GetGroupConversation(groupID string, limit, offset int) ([]models.Message, int64, error) {
	var messages []models.Message
	var count int64

	// Get total count of messages in group
	r.db.Model(&models.Message{}).
		Where("group_id = ?", groupID).
		Count(&count)

	// Get the messages with limit and offset.
	err := r.db.
		Where("group_id = ?", groupID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, count, err
}
