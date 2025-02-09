package websockets

// MessageSaver is an interface for saving messages.
type MessageSaver interface {
	SendMessage(senderID, receiverID, groupID, content string) error // Corrected signature
}
