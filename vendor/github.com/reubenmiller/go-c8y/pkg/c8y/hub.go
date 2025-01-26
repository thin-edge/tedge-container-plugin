package c8y

import "sync"

// Hub maintains the set of active subscriptions and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[*subscription]bool

	// Inbound messages from the clients.
	broadcast chan *Message

	// Register requests from the clients.
	register chan *subscription

	// Unregister requests from clients by channel name.
	unregister chan string

	// Return a channel of channels
	getChannels chan chan string

	channels sync.Map
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *Message),
		register:   make(chan *subscription),
		unregister: make(chan string),
		clients:    make(map[*subscription]bool),

		getChannels: make(chan chan string),
	}
}

// GetActiveChannels returns the list of active channels which are currently subscribed to
func (h *Hub) GetActiveChannels() []string {
	channels := []string{}
	h.channels.Range(func(key, value interface{}) bool {
		channels = append(channels, key.(string))
		return true
	})
	return channels
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.channels.Store(client.glob.String(), true)
		case channel := <-h.unregister:
			// Delete clients with the same channel pattern
			h.channels.Delete(channel)
			for client := range h.clients {
				if client.glob.String() == channel {
					delete(h.clients, client)
				}
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				if ((client.isWildcard && client.glob.MatchString(message.Channel)) || client.glob.String() == message.Channel) && !client.disabled {
					client.out <- message
				}
			}
		}
	}
}
