package websockets

// MessageSaver is an interface for saving messages.
type MessageSaver interface {
	SendMessage(senderID, receiverID, groupID, content, replyToMessageID, fileName, filePath, fileType string, fileSize int64, checksum string) (string, error)
	AddReaction(messageID, userID, emoji string) error
	RemoveReaction(messageID, userID, emoji string) error
	UpdateMessageStatus(messageID string, status string) error
}
