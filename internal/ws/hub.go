package ws

import "sync"

// registration pairs a client with the task ID it wants to watch.
type registration struct {
	client *Client
	taskID int64
}

// Message carries a payload destined for every client watching a given task.
type Message struct {
	TaskID int64
	Data   []byte
}

// Hub maintains the set of active clients and broadcasts messages to clients
// grouped by task ID.
type Hub struct {
	mu         sync.Mutex
	clients    map[int64]map[*Client]bool
	register   chan *registration
	unregister chan *Client
	broadcast  chan *Message
}

// NewHub creates a ready-to-run Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int64]map[*Client]bool),
		register:   make(chan *registration),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 256),
	}
}

// Run processes register, unregister, and broadcast events. It should be
// started as a goroutine and will run until the channels are closed.
func (h *Hub) Run() {
	for {
		select {
		case reg := <-h.register:
			h.mu.Lock()
			if h.clients[reg.taskID] == nil {
				h.clients[reg.taskID] = make(map[*Client]bool)
			}
			h.clients[reg.taskID][reg.client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			taskID := client.taskID
			if conns, ok := h.clients[taskID]; ok {
				if _, exists := conns[client]; exists {
					delete(conns, client)
					close(client.send)
					if len(conns) == 0 {
						delete(h.clients, taskID)
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.Lock()
			if conns, ok := h.clients[msg.TaskID]; ok {
				for client := range conns {
					select {
					case client.send <- msg.Data:
					default:
						// Client's send buffer is full; drop it.
						delete(conns, client)
						close(client.send)
						if len(conns) == 0 {
							delete(h.clients, msg.TaskID)
						}
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

// Register adds a client to the broadcast group for the given task.
func (h *Hub) Register(taskID int64, client *Client) {
	h.register <- &registration{client: client, taskID: taskID}
}

// Unregister removes a client from its task broadcast group.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends data to every client watching the specified task.
func (h *Hub) Broadcast(taskID int64, data []byte) {
	h.broadcast <- &Message{TaskID: taskID, Data: data}
}
