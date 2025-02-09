package websockets

// MessageSaver is an interface for saving messages.  This breaks the circular dependency.
type MessageSaver interface {
	SendMessage(senderID, receiverID, content string) error
}
