// websockets/hub.go
package websockets

import (
	"encoding/json"
	"log"
)

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	// Registered clients.  Key is the UserID.
	Clients map[string]*Client

	// Inbound messages from the clients.
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
		Broadcast:  make(chan []byte),
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
		case client := <-h.Unregister:
			if _, ok := h.Clients[client.UserID]; ok {
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
			}
		case message := <-h.Broadcast:
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err == nil {
				if groupID, ok := msg["group_id"].(string); ok && groupID != "" {
					log.Printf("Broadcasting group message to group %s", groupID)
					// Group message: send only to members of the group
					if members, ok := h.Groups[groupID]; ok {
						log.Printf("Found %d members in group %s", len(members), groupID)
						for userID := range members {
							if client, ok := h.Clients[userID]; ok {
								select {
								case client.Send <- message:
									log.Printf("Sent message to user %s", userID)
								default:
									close(client.Send)
									delete(h.Clients, client.UserID)
									delete(members, userID)
									log.Printf("Removed inactive user %s from group", userID)
								}
							}
						}
					} else {
						log.Printf("No members found for group %s", groupID)
					}
					continue
				}
			}
			// Default case (direct message, or malformed group message): broadcast to all
			for _, client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					// If the client's send channel is full, assume they're disconnected.
					close(client.Send)
					delete(h.Clients, client.UserID)
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
