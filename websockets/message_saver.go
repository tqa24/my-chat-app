package websockets

// MessageSaver is an interface for saving messages.
type MessageSaver interface {
	SendMessage(senderID, receiverID, groupID, content string, replyToMessageID string) error
	AddReaction(messageID, userID, emoji string) error
	RemoveReaction(messageID, userID, emoji string) error
}
