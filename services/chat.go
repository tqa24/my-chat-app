package services

import (
	"fmt"
	"my-chat-app/models"
	"my-chat-app/repositories"
	"my-chat-app/websockets"
	"strconv"

	"github.com/google/uuid"
)

type ChatService interface {
	SendMessage(senderID, receiverID, content string) error // Return only error
	GetConversation(user1ID, user2ID string, pageStr, pageSizeStr string) ([]models.Message, error)
	UpdateMessageStatus(messageID string, status string) error
}

type chatService struct {
	messageRepo repositories.MessageRepository
	hub         *websockets.Hub // Inject the WebSocket hub
}

func NewChatService(messageRepo repositories.MessageRepository, hub *websockets.Hub) ChatService {
	return &chatService{messageRepo, hub}
}

func (s *chatService) SendMessage(senderID, receiverID, content string) error { // Return only error
	senderUUID, err := uuid.Parse(senderID)
	if err != nil {
		return fmt.Errorf("invalid sender ID: %v", err)
	}
	receiverUUID, err := uuid.Parse(receiverID)
	if err != nil {
		return fmt.Errorf("invalid receiver ID: %v", err)
	}
	message := &models.Message{
		SenderID:   senderUUID,
		ReceiverID: receiverUUID,
		Content:    content,
		Status:     "sent", // Initial status
	}

	err = s.messageRepo.Create(message)
	if err != nil {
		return err
	}
	// Broadcast the message to the receiver via WebSockets
	s.hub.Broadcast <- []byte(fmt.Sprintf(`{"type": "new_message", "sender_id": "%s", "receiver_id": "%s", "content": "%s", "message_id": "%s", "created_at": "%s"}`, senderID, receiverID, content, message.ID.String(), message.CreatedAt.Format("2006-01-02 15:04:05")))

	return nil // Return only the error
}
func (s *chatService) GetConversation(user1ID, user2ID string, pageStr, pageSizeStr string) ([]models.Message, error) {
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10 // Default page size
	}

	offset := (page - 1) * pageSize

	return s.messageRepo.GetConversation(user1ID, user2ID, pageSize, offset)
}
func (s *chatService) UpdateMessageStatus(messageID string, status string) error {
	//TODO:
	return nil
}
