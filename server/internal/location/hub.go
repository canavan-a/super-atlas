package location

import "sync"

// Hub tracks which users currently have an open WebSocket connection so
// location updates can be pushed to them in real time. A user may have more
// than one open connection (multiple devices), so each maps to a set.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Register(userID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][c] = struct{}{}
}

func (h *Hub) Unregister(userID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients[userID], c)
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
}

// PushTo sends a payload to every open connection belonging to userID.
func (h *Hub) PushTo(userID string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		c.Send(payload)
	}
}
