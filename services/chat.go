package services

import (
	"encoding/json"
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
		GroupID:    groupUUID,
		Content:    content,
		Status:     "sent",
	}

	err = s.messageRepo.Create(message)
	if err != nil {
		return err
	}

	// Construct a map for the message data
	msgData := map[string]interface{}{
		"type":       "new_message",
		"sender_id":  senderID,
		"content":    content,
		"message_id": message.ID.String(),
		"created_at": message.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	// Add receiver_id or group_id based on message type
	if groupUUID != nil {
		msgData["group_id"] = groupID
		// Group message:  Broadcast to group members *only*.
		// Loop through all users of this group
		for userID := range s.hub.Groups[groupID] {
			if client, ok := s.hub.Clients[userID]; ok {
				msgBytes, _ := json.Marshal(msgData) // Convert to JSON
				select {
				case client.Send <- msgBytes: // Send the JSON message
				default:
					close(client.Send)
					delete(s.hub.Clients, userID)
				}
			}
		}
	} else if receiverUUID != nil {
		msgData["receiver_id"] = receiverID
		msgBytes, _ := json.Marshal(msgData)
		s.hub.Broadcast <- msgBytes // Send to specific user
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
