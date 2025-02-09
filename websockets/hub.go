package websockets

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
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[string]*Client),
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
			}
		case message := <-h.Broadcast:
			// Iterate through ALL clients (inefficient - will optimize later).
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
