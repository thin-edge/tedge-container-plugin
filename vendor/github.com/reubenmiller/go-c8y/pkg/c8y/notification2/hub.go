package notification2

import "bytes"

const (
	SourceWildcard = "*"
	ActionWildcard = "*"
)

// Hub maintains the set of active subscriptions and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[*ClientSubscription]bool

	// Inbound messages from the clients.
	broadcast chan Message

	// Register requests from the clients.
	register chan *ClientSubscription

	// Unregister requests from clients by channel name.
	unregister chan *ClientSubscription
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan Message),
		register:   make(chan *ClientSubscription),
		unregister: make(chan *ClientSubscription),
		clients:    make(map[*ClientSubscription]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case subscription := <-h.unregister:
			// Delete clients with the same channel pattern
			for client := range h.clients {
				if subscription.Pattern == SourceWildcard || subscription.Pattern == client.Pattern {
					delete(h.clients, client)
				}
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				matchesPattern := client.Pattern == "" || client.Pattern == SourceWildcard || bytes.Contains(message.Description, []byte(client.Pattern))
				matchesAction := client.Action == "" || client.Action != "" && (client.Action == ActionWildcard || bytes.Contains(message.Action, []byte(client.Action)))

				if matchesPattern && matchesAction {
					client.Out <- message
				}
			}
		}
	}
}
