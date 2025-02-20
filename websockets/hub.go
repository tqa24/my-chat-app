package websockets

import (
	"encoding/json"
	"log"
)

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	// Registered clients.  Key is the UserID.
	Clients map[string]*Client

	// Inbound messages from the clients.  DEPRECATED.
	Broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client

	// Group memberships.  Key is groupID, value is a set of userIDs.
	Groups map[string]map[string]bool // Add this
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte), // Keep this for now, but it's deprecated
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[string]*Client),
		Groups:     make(map[string]map[string]bool), // Initialize Groups
	}
}
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.UserID] = client // Register by UserID
			// Broadcast online status when client registers
			statusMsg := []byte(`{"type": "online_status", "user_id": "` + client.UserID + `"}`)
			for _, c := range h.Clients {
				select {
				case c.Send <- statusMsg:
				default:
					close(c.Send)
					delete(h.Clients, c.UserID)
				}
			}
			log.Printf("Client registered: %s", client.UserID)

		case client := <-h.Unregister:
			if _, ok := h.Clients[client.UserID]; ok {
				// Broadcast offline status before removing client
				statusMsg := []byte(`{"type": "offline_status", "user_id": "` + client.UserID + `"}`)
				for _, c := range h.Clients {
					select {
					case c.Send <- statusMsg:
					default:
						close(c.Send)
						delete(h.Clients, c.UserID)
					}
				}
				delete(h.Clients, client.UserID)
				close(client.Send)
				// Remove the client from all groups
				for groupID, members := range h.Groups {
					if _, ok := members[client.UserID]; ok {
						delete(members, client.UserID)
						// If the group is now empty, delete it
						if len(members) == 0 {
							delete(h.Groups, groupID)
						}
					}
				}
				log.Printf("Client unregistered: %s", client.UserID)
			}
		case message := <-h.Broadcast: //Handle broadcast
			// Check if it's a group message by attempting to unmarshal it and checking for group_id
			var msg map[string]interface{}
			//Try to unmarshal
			if err := json.Unmarshal(message, &msg); err == nil {
				msgType, _ := msg["type"].(string)

				// Handle status messages differently
				if msgType == "online_status" || msgType == "offline_status" {
					// Broadcast status to all clients
					for _, client := range h.Clients {
						select {
						case client.Send <- message:
						default:
							close(client.Send)
							delete(h.Clients, client.UserID)
						}
					}
					continue
				}

				// Handle reaction messages
				if msgType == "reaction_added" || msgType == "reaction_removed" {
					// Check group_id
					if groupID, ok := msg["group_id"].(string); ok && groupID != "" {
						// Group message: send only to members of the group
						if members, ok := h.Groups[groupID]; ok {
							for userID := range members {
								if client, ok := h.Clients[userID]; ok {
									select {
									case client.Send <- message: // Send to the client
									default:
										// If the client's send channel is full, assume they're disconnected.
										close(client.Send)
										delete(h.Clients, client.UserID)
										// Remove from group as well
										delete(members, userID)
									}
								}
							}
							// If the group is now empty, delete it
							if len(members) == 0 {
								delete(h.Groups, groupID)
							}
						}
						continue // Important: Skip the default broadcast
						// If not group message, it mean that this message is direct message
					} else {
						// Get receiverID from message
						if receiverID, ok := msg["receiver_id"].(string); ok && receiverID != "" {
							if client, ok := h.Clients[receiverID]; ok { // Check client exist
								select {
								case client.Send <- message:
								default:
									close(client.Send)
									delete(h.Clients, receiverID)
								}
							}
						}
						// Also send back to sender
						if senderID, ok := msg["sender_id"].(string); ok && senderID != "" {
							if client, ok := h.Clients[senderID]; ok {
								select {
								case client.Send <- message:
								default:
									close(client.Send)
									delete(h.Clients, senderID)
								}
							}
						}
					}
					continue // Skip further process
				}
				//Check group_id
				//if groupID, ok := msg["group_id"].(string); ok && groupID != "" {  // Removed the unnecessary type-specific check
				if groupID, ok := msg["group_id"].(string); ok && groupID != "" {
					// Group message: send only to members of the group
					if members, ok := h.Groups[groupID]; ok {
						for userID := range members {
							if client, ok := h.Clients[userID]; ok {
								select {
								case client.Send <- message: // Send to the client
								default:
									// If the client's send channel is full, assume they're disconnected.
									close(client.Send)
									delete(h.Clients, client.UserID)
									// Remove from group as well
									delete(members, userID)
								}
							}
						}
						// If the group is now empty, delete it
						if len(members) == 0 {
							delete(h.Groups, groupID)
						}
					}
					continue // Skip the default broadcast
					// If not group message, it mean that this message is direct message
				} else { // Direct message handling
					// Get receiverID from message
					if receiverID, ok := msg["receiver_id"].(string); ok && receiverID != "" {
						if client, ok := h.Clients[receiverID]; ok { // Check client exist
							select {
							case client.Send <- message:
							default:
								close(client.Send)
								delete(h.Clients, receiverID)
							}
						}
					}
					// Also send back to sender
					if senderID, ok := msg["sender_id"].(string); ok && senderID != "" {
						if client, ok := h.Clients[senderID]; ok {
							select {
							case client.Send <- message:
							default:
								close(client.Send)
								delete(h.Clients, senderID)
							}
						}
					}
				}
			}
		}
	}
}

// AddClientToGroup adds a client (by UserID) to a group.
func (h *Hub) AddClientToGroup(userID, groupID string) {
	log.Printf("Hub add client to group: %v %v", userID, groupID)
	if _, ok := h.Groups[groupID]; !ok {
		h.Groups[groupID] = make(map[string]bool)
	}
	h.Groups[groupID][userID] = true
}

// RemoveClientFromGroup removes a client (by UserID) from a group.
func (h *Hub) RemoveClientFromGroup(userID, groupID string) {
	if _, ok := h.Groups[groupID]; ok {
		delete(h.Groups[groupID], userID)
		// If the group is now empty, delete it.
		if len(h.Groups[groupID]) == 0 {
			delete(h.Groups, groupID)
		}
	}
}

// GetGroupMembers gets all UserIDs in a group.
func (h *Hub) GetGroupMembers(groupID string) []string {
	members := []string{}
	if _, ok := h.Groups[groupID]; ok {
		for userID := range h.Groups[groupID] {
			members = append(members, userID)
		}
	}

	return members
}
