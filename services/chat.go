package services

import (
	"encoding/json"
	"fmt"
	"my-chat-app/models"
	"my-chat-app/repositories"
	"my-chat-app/websockets"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ChatService interface {
	SendMessage(senderID, receiverID, groupID, content string, replyToMessageID string) error
	GetConversation(user1ID, user2ID string, pageStr, pageSizeStr string) ([]models.Message, int64, error)
	GetGroupConversation(groupID string, pageStr, pageSizeStr string) ([]models.Message, int64, error)
	UpdateMessageStatus(messageID string, status string) error
	AddReaction(messageID, userID, reaction string) error    // NEW
	RemoveReaction(messageID, userID, reaction string) error // NEW
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

func (s *chatService) SendMessage(senderID, receiverID, groupID, content string, replyToMessageID string) error {
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
	// Handle Reply-to
	var replyToUUID *uuid.UUID
	if replyToMessageID != "" {
		replyID, err := uuid.Parse(replyToMessageID)
		if err != nil {
			return fmt.Errorf("invalid reply_to_message_id: %v", err)
		}
		replyToUUID = &replyID
	}

	message := &models.Message{
		SenderID:         senderUUID,
		ReceiverID:       receiverUUID,
		GroupID:          groupUUID,
		Content:          content,
		Status:           "sent",
		ReplyToMessageID: replyToUUID, // Set ReplyToMessageID
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
	// Add reply_to_message_id if present
	if replyToUUID != nil {
		msgData["reply_to_message_id"] = replyToMessageID
		// Get the original message to include its content
		if originalMsg, err := s.messageRepo.GetByID(replyToMessageID); err == nil {
			msgData["reply_to_message"] = map[string]interface{}{
				"id":        originalMsg.ID.String(),
				"content":   originalMsg.Content,
				"sender_id": originalMsg.SenderID.String(),
			}
		}
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

func (s *chatService) GetConversation(user1ID, user2ID string, pageStr, pageSizeStr string) ([]models.Message, int64, error) {
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

func (s *chatService) GetGroupConversation(groupID string, pageStr, pageSizeStr string) ([]models.Message, int64, error) {
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10 // Default page size
	}

	offset := (page - 1) * pageSize

	return s.messageRepo.GetGroupConversation(groupID, pageSize, offset) // Return count as well
}
func (s *chatService) UpdateMessageStatus(messageID string, status string) error {
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return err
	}

	message.Status = status
	return s.messageRepo.Update(message)
}

// AddReaction adds a reaction to a message.
func (s *chatService) AddReaction(messageID, userID, reaction string) error {
	_, err := uuid.Parse(messageID)
	if err != nil {
		return fmt.Errorf("invalid message ID: %v", err)
	}
	_, err = uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %v", err)
	}

	message, err := s.messageRepo.GetByID(messageID) // You need a GetByID in your repo
	if err != nil {
		return err
	}
	if message == nil {
		return fmt.Errorf("message not found")
	}

	// Update the reactions.  We're using a map where keys are reactions,
	// and values are arrays of user IDs.
	var reactions map[string][]string
	if err := json.Unmarshal(message.Reactions, &reactions); err != nil {
		// If unmarshaling fails, assume empty reactions
		reactions = make(map[string][]string)
	}

	// Check if the user has already reacted with this emoji.
	userList := reactions[reaction]
	for _, u := range userList {
		if u == userID {
			return fmt.Errorf("user has already reacted with this emoji")
		}
	}

	reactions[reaction] = append(reactions[reaction], userID)
	//Marshal back
	updatedReactions, err := json.Marshal(reactions)
	if err != nil {
		return fmt.Errorf("error when marshal reactions")
	}
	message.Reactions = datatypes.JSON(updatedReactions)
	return s.messageRepo.Update(message) // You need an Update method
}

// RemoveReaction removes a reaction from a message.
func (s *chatService) RemoveReaction(messageID, userID, reaction string) error {
	_, err := uuid.Parse(messageID)
	if err != nil {
		return fmt.Errorf("invalid message ID: %v", err)
	}
	_, err = uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %v", err)
	}

	message, err := s.messageRepo.GetByID(messageID) // Assuming you have GetByID
	if err != nil {
		return err
	}
	if message == nil {
		return fmt.Errorf("message not found")
	}

	var reactions map[string][]string
	if err := json.Unmarshal(message.Reactions, &reactions); err != nil {
		// If it's completely broken, just return (nothing to remove)
		return nil
	}

	userList, ok := reactions[reaction]
	if !ok {
		return nil // Reaction doesn't exist, nothing to do
	}

	// Remove the user from the list.
	for i, u := range userList {
		if u == userID {
			reactions[reaction] = append(userList[:i], userList[i+1:]...)
			break
		}
	}

	// If the reaction list is now empty, remove the reaction key.
	if len(reactions[reaction]) == 0 {
		delete(reactions, reaction)
	}

	updatedReactions, err := json.Marshal(reactions)
	if err != nil {
		return fmt.Errorf("error when marshal reactions")
	}
	message.Reactions = datatypes.JSON(updatedReactions)
	return s.messageRepo.Update(message) // Update the message in the repo
}
