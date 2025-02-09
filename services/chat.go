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
	SendMessage(senderID, receiverID, groupID, content string) error // Updated signature
	GetConversation(user1ID, user2ID string, pageStr, pageSizeStr string) ([]models.Message, error)
	GetGroupConversation(groupID string, pageStr, pageSizeStr string) ([]models.Message, error)
	UpdateMessageStatus(messageID string, status string) error
}

type chatService struct {
	messageRepo repositories.MessageRepository
	groupRepo   repositories.GroupRepository // Inject GroupRepository
	hub         *websockets.Hub              // Inject the WebSocket hub
}

// Update NewChatService to accept GroupRepository
func NewChatService(messageRepo repositories.MessageRepository, groupRepo repositories.GroupRepository, hub *websockets.Hub) ChatService {
	return &chatService{messageRepo, groupRepo, hub}
}

func (s *chatService) SendMessage(senderID, receiverID, groupID, content string) error {
	senderUUID, err := uuid.Parse(senderID)
	if err != nil {
		return fmt.Errorf("invalid sender ID: %v", err)
	}

	var receiverUUID *uuid.UUID
	if receiverID != "" {
		id, err := uuid.Parse(receiverID)
		if err != nil {
			return fmt.Errorf("invalid receiver ID: %v", err)
		}
		receiverUUID = &id
	}

	var groupUUID *uuid.UUID
	if groupID != "" {
		id, err := uuid.Parse(groupID)
		if err != nil {
			return fmt.Errorf("invalid group ID: %v", err)
		}
		groupUUID = &id
	}

	message := &models.Message{
		SenderID:   senderUUID,
		ReceiverID: receiverUUID,
		GroupID:    groupUUID, // Set the GroupID
		Content:    content,
		Status:     "sent", // Initial status
	}

	err = s.messageRepo.Create(message)
	if err != nil {
		return err
	}

	// Broadcast logic (will be updated later for groups)
	if groupUUID != nil {
		// Group message. Get all user and send message
		users, err := s.groupRepo.GetUsers(&models.Group{ID: *groupUUID})
		if err != nil {
			return err
		}
		for _, user := range users {
			s.hub.Broadcast <- []byte(fmt.Sprintf(`{"type": "new_message", "sender_id": "%s", "group_id": "%s", "content": "%s", "message_id": "%s", "created_at": "%s", "receiver_id": "%s"}`, senderID, groupID, content, message.ID.String(), message.CreatedAt.Format("2006-01-02 15:04:05"), user.ID.String()))

		}

	} else {
		// Direct message
		s.hub.Broadcast <- []byte(fmt.Sprintf(`{"type": "new_message", "sender_id": "%s", "receiver_id": "%s", "content": "%s", "message_id": "%s", "created_at": "%s"}`, senderID, receiverID, content, message.ID.String(), message.CreatedAt.Format("2006-01-02 15:04:05")))
	}

	return nil
}

func (s *chatService) GetConversation(user1ID, user2ID string, pageStr, pageSizeStr string) ([]models.Message, error) {
	// ... (existing GetConversation implementation - no changes needed here) ...
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
func (s *chatService) GetGroupConversation(groupID string, pageStr, pageSizeStr string) ([]models.Message, error) {
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10 // Default page size
	}

	offset := (page - 1) * pageSize
	return s.messageRepo.GetGroupConversation(groupID, pageSize, offset)
}
func (s *chatService) UpdateMessageStatus(messageID string, status string) error {
	//TODO:
	return nil
}
