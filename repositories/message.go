package repositories

import (
	"gorm.io/gorm"
	"my-chat-app/models"
)

type MessageRepository interface {
	Create(message *models.Message) error
	GetConversation(user1ID, user2ID string, limit, offset int) ([]models.Message, int64, error) // Return messages and total count
	GetGroupConversation(groupID string, limit, offset int) ([]models.Message, int64, error)     // Return messages and total count
	GetByID(id string) (*models.Message, error)                                                  // NEW
	Update(message *models.Message) error                                                        // NEW
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db}
}

func (r *messageRepository) Create(message *models.Message) error {
	result := r.db.Create(message)
	//if result.Error != nil {
	//	// If it's a duplicate file error, we can ignore it
	//	if strings.Contains(result.Error.Error(), "idx_file_checksum") {
	//		return nil
	//	}
	//	return result.Error
	//}
	return result.Error
}

func (r *messageRepository) GetConversation(user1ID, user2ID string, limit, offset int) ([]models.Message, int64, error) {
	var messages []models.Message
	var count int64

	// Get the total count of messages
	r.db.Model(&models.Message{}).
		Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)", user1ID, user2ID, user2ID, user1ID).
		Count(&count)

	// Get the messages with limit, offset, and preloading of ReplyToMessage
	err := r.db.
		Preload("ReplyToMessage"). // Preload the ReplyToMessage
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

	// Get the messages with limit, offset and preloading of ReplyToMessage.
	err := r.db.
		Preload("ReplyToMessage"). // Preload the ReplyToMessage
		Where("group_id = ?", groupID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, count, err
}

func (r *messageRepository) GetByID(id string) (*models.Message, error) {
	var message models.Message
	err := r.db.Where("id = ?", id).First(&message).Error
	return &message, err
}

func (r *messageRepository) Update(message *models.Message) error {
	return r.db.Save(message).Error
}
