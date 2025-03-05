package services

import (
	"encoding/json"
	"fmt"
	"log"
	"my-chat-app/models"
	"my-chat-app/repositories"
	"my-chat-app/websockets"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AIUserID is a constant for the AI Assistant's user ID.
const AIUserID = "00000000-0000-0000-0000-000000000000"

type ChatService interface {
	SendMessage(senderID, receiverID, groupID, content, replyToMessageID, fileName, filePath, fileType string, fileSize int64, checksum string) (string, error)
	SendMessageForWebSocket(senderID, receiverID, groupID, content, replyToMessageID string) error
	GetConversation(user1ID, user2ID string, pageStr, pageSizeStr string) ([]models.Message, int64, error)
	GetGroupConversation(groupID string, pageStr, pageSizeStr string) ([]models.Message, int64, error)
	UpdateMessageStatus(messageID string, status string) error
	AddReaction(messageID, userID, reaction string) error
	RemoveReaction(messageID, userID, reaction string) error
}

type chatService struct {
	messageRepo repositories.MessageRepository
	groupRepo   repositories.GroupRepository
	userRepo    repositories.UserRepository
	hub         *websockets.Hub
	aiService   AIService
}

func NewChatService(messageRepo repositories.MessageRepository, groupRepo repositories.GroupRepository, userRepo repositories.UserRepository, hub *websockets.Hub, aiService AIService) ChatService {
	return &chatService{messageRepo, groupRepo, userRepo, hub, aiService}
}

func (s *chatService) SendMessageForWebSocket(senderID, receiverID, groupID, content, replyToMessageID string) error {
	// Call the *full* SendMessage, with default values for file-related parameters.
	_, err := s.SendMessage(senderID, receiverID, groupID, content, replyToMessageID, "", "", "", 0, "")
	return err
}

func (s *chatService) SendMessage(senderID, receiverID, groupID, content, replyToMessageID, fileName, filePath, fileType string, fileSize int64, checksum string) (string, error) {
	// Check content size
	const maxContentSize = 8192 // 8KB
	if len(content) > maxContentSize {
		return "", fmt.Errorf("message content exceeds maximum size limit")
	}

	log.Printf("chatService.SendMessage: senderID=%s, receiverID=%s, groupID=%s, content=%s, replyToMessageID=%s, fileName=%s, filePath=%s, fileType=%s, fileSize=%d, checksum=%s",
		senderID, receiverID, groupID, content, replyToMessageID, fileName, filePath, fileType, fileSize, checksum)

	senderUUID, err := uuid.Parse(senderID)
	if err != nil {
		return "", fmt.Errorf("invalid sender ID: %v", err)
	}

	var receiverUUID *uuid.UUID
	if receiverID != "" {
		id, err := uuid.Parse(receiverID)
		if err != nil {
			return "", fmt.Errorf("invalid receiver ID: %v", err)
		}
		receiverUUID = &id
	}

	var groupUUID *uuid.UUID
	if groupID != "" {
		id, err := uuid.Parse(groupID)
		if err != nil {
			return "", fmt.Errorf("invalid group ID: %v", err)
		}
		groupUUID = &id
	}
	// Handle Reply-to
	var replyToUUID *uuid.UUID
	if replyToMessageID != "" {
		replyID, err := uuid.Parse(replyToMessageID)
		if err != nil {
			return "", fmt.Errorf("invalid reply_to_message_id: %v", err)
		}
		replyToUUID = &replyID
	}

	// --- DIRECT AI MESSAGE HANDLING ---
	isDirectAIMessage := receiverID == AIUserID

	var aiResponse string
	if isDirectAIMessage {
		// Immediately process the message with the AI service.
		aiResponse, err = s.aiService.ProcessMessage(content) // No need for @AI now
		if err != nil {
			log.Printf("Error processing AI message: %v", err)
			aiResponse = "Sorry, I couldn't process your request."
		}
	} else if strings.Contains(content, "@AI") {
		// Check for AI mention *before* saving the original message.
		aiResponse, err = s.aiService.ProcessMessage(content)
		if err != nil {
			log.Printf("Error processing AI message: %v", err)
			//  Return the original message and add an error
			aiResponse = "Sorry, I couldn't process your request."
		}
	}
	// --- END DIRECT AI MESSAGE HANDLING ---

	// Create the user's message (always create this).
	userMessage := &models.Message{
		SenderID:         senderUUID,
		ReceiverID:       receiverUUID,
		GroupID:          groupUUID,
		Content:          content, // Original user message content
		Status:           "sent",
		ReplyToMessageID: replyToUUID,
		FileName:         fileName,
		FilePath:         filePath,
		FileType:         fileType,
		FileSize:         fileSize,
		FileChecksum:     checksum,
	}

	//Save User message to DB.
	err = s.messageRepo.Create(userMessage)
	if err != nil {
		return "", err
	}
	// Get Sender Username.
	senderUser, err := s.userRepo.GetByID(senderID)
	if err != nil {
		fmt.Printf("Error fetching sender user: %v\n", err)
	}

	// Prepare the user message data for broadcasting.
	userMsgData := map[string]interface{}{
		"type":            "new_message",
		"sender_id":       senderID,
		"sender_username": senderUser.Username,
		"content":         content,
		"message_id":      userMessage.ID.String(),
		"created_at":      userMessage.CreatedAt.Format("2006-01-02 15:04:05"),
		"file_name":       fileName,
		"file_path":       filePath,
		"file_type":       fileType,
		"file_size":       fileSize,
	}
	if replyToUUID != nil {
		userMsgData["reply_to_message_id"] = replyToMessageID
		if originalMsg, err := s.messageRepo.GetByID(replyToMessageID); err == nil {
			userMsgData["reply_to_message"] = map[string]interface{}{
				"id":        originalMsg.ID.String(),
				"content":   originalMsg.Content,
				"sender_id": originalMsg.SenderID.String(),
			}
		}
	}

	//Add receiver_id and group_id to message data.
	if groupUUID != nil {
		userMsgData["group_id"] = groupID
	} else if receiverUUID != nil {
		userMsgData["receiver_id"] = receiverID
	}

	// ---  BROADCAST USER MESSAGE ---
	userMsgBytes, _ := json.Marshal(userMsgData)
	log.Printf("Consumer about to broadcast: %s", string(userMsgBytes))
	if groupUUID != nil {
		// Group message:  Broadcast to group members.
		for userID := range s.hub.Groups[groupID] {
			if client, ok := s.hub.Clients[userID]; ok {
				client.Send <- userMsgBytes // Send to each member
			}
		}
	} else if receiverUUID != nil {
		// Direct Message: Send to *BOTH* sender and receiver.
		if senderClient, ok := s.hub.Clients[senderID]; ok {
			senderClient.Send <- userMsgBytes
		}
		if receiverClient, ok := s.hub.Clients[receiverID]; ok {
			receiverClient.Send <- userMsgBytes
		}
	}
	// --- END BROADCAST USER MESSAGE ---

	// --- AI RESPONSE HANDLING (Both Direct and Mentions) ---
	if isDirectAIMessage || strings.Contains(content, "@AI") { // Handle both cases
		aiSenderUUID := uuid.MustParse(AIUserID) // AI's UUID

		// For direct messages, set receiver to original sender
		var aiReceiverUUID *uuid.UUID
		if groupUUID == nil {
			// In direct messages, AI responds to the original sender
			aiReceiverUUID = &senderUUID
		} else {
			// In group messages, keep the same group context
			aiReceiverUUID = receiverUUID
		}

		aiMessage := &models.Message{
			SenderID:         aiSenderUUID, // AI is the sender
			ReceiverID:       aiReceiverUUID,
			GroupID:          groupUUID,  // Same group as the original message
			Content:          aiResponse, // The AI's generated response
			Status:           "sent",
			ReplyToMessageID: &userMessage.ID, // Reply to the *user's* message
		}

		if err := s.messageRepo.Create(aiMessage); err != nil {
			return "", err
		}

		// Prepare AI message for broadcast.
		aiMsgData := map[string]interface{}{
			"type":                "new_message",
			"sender_id":           AIUserID,       // Clearly indicate AI sender
			"sender_username":     "AI_Assistant", // Set sender username for AI
			"message_id":          aiMessage.ID.String(),
			"content":             aiResponse,
			"created_at":          aiMessage.CreatedAt.Format("2006-01-02 15:04:05"),
			"reply_to_message_id": userMessage.ID.String(), // Reply to the user's message
			// Include reply message data
			"reply_to_message": map[string]interface{}{
				"id":        userMessage.ID.String(),
				"content":   userMessage.Content,
				"sender_id": userMessage.SenderID.String(),
			},
		}
		if groupUUID != nil {
			aiMsgData["group_id"] = groupID
		} else if receiverUUID != nil {
			aiMsgData["receiver_id"] = receiverID
		}

		// --- BROADCAST AI RESPONSE ---
		aiMsgBytes, _ := json.Marshal(aiMsgData)
		if groupUUID != nil {
			// Group message: send to all group members
			for userID := range s.hub.Groups[groupID] {
				if client, ok := s.hub.Clients[userID]; ok {
					client.Send <- aiMsgBytes
				}
			}
		} else if receiverUUID != nil {
			// Direct Message:  Send to *BOTH* sender and receiver.
			if senderClient, ok := s.hub.Clients[senderID]; ok {
				senderClient.Send <- aiMsgBytes
			}
			if receiverClient, ok := s.hub.Clients[receiverID]; ok {
				receiverClient.Send <- aiMsgBytes
			}
		}
		// --- END BROADCAST AI RESPONSE ---
		return aiMessage.ID.String(), nil // Return AI message ID for consistency
	}
	// --- END AI RESPONSE HANDLING ---

	return userMessage.ID.String(), nil // Return the original message's ID.
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

	// Update the reactions. Using a map where keys are reactions,
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

	// *** BROADCAST ADDED REACTION ***
	broadcastMessage := map[string]interface{}{
		"type":       "reaction_added",
		"message_id": messageID,
		"user_id":    userID,
		"emoji":      reaction,
		// NO group_id here initially
	}

	// Add group_id ONLY if it's a group message
	if message.GroupID != nil {
		broadcastMessage["group_id"] = message.GroupID.String()
	}

	broadcastBytes, _ := json.Marshal(broadcastMessage)

	// Determine who to broadcast to (group or specific user)
	if message.GroupID != nil {
		// Iterate through the group members in the hub.
		for memberUserID := range s.hub.Groups[message.GroupID.String()] {
			if client, ok := s.hub.Clients[memberUserID]; ok {
				select {
				case client.Send <- broadcastBytes: // Send to each member
				default:
					close(client.Send)
					delete(s.hub.Clients, memberUserID) // Clean up
				}
			}
		}
	} else if message.ReceiverID != nil {
		// It's a direct message, broadcast to the sender and receiver
		if client, ok := s.hub.Clients[message.ReceiverID.String()]; ok {
			client.Send <- broadcastBytes
		}
		// Also send back to sender
		if client, ok := s.hub.Clients[message.SenderID.String()]; ok {
			client.Send <- broadcastBytes
		}
	}
	err = s.messageRepo.Update(message)
	if err != nil {
		return err
	}
	return nil // Return the result of the database update
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

	// *** BROADCAST REMOVED REACTION ***
	broadcastMessage := map[string]interface{}{
		"type":       "reaction_removed",
		"message_id": messageID,
		"user_id":    userID,
		"emoji":      reaction,
		// NO group_id here initially
	}

	// Add group_id ONLY if it's a group message
	if message.GroupID != nil {
		broadcastMessage["group_id"] = message.GroupID.String()
	}

	broadcastBytes, _ := json.Marshal(broadcastMessage)

	// Determine who to broadcast to (group or specific user)
	if message.GroupID != nil {
		// Iterate through the group members in the hub.
		for memberUserID := range s.hub.Groups[message.GroupID.String()] {
			if client, ok := s.hub.Clients[memberUserID]; ok {
				select {
				case client.Send <- broadcastBytes:
				default:
					close(client.Send)
					delete(s.hub.Clients, memberUserID)
				}
			}
		}
	} else if message.ReceiverID != nil {
		// It's a direct message, broadcast to the sender and receiver
		if client, ok := s.hub.Clients[message.ReceiverID.String()]; ok {
			client.Send <- broadcastBytes
		}
		//Also send back to sender
		if client, ok := s.hub.Clients[message.SenderID.String()]; ok {
			client.Send <- broadcastBytes
		}
	}
	err = s.messageRepo.Update(message) // Update the message in the repo
	if err != nil {
		return err
	}
	return nil
}
