package location

import (
	"log"

	"github.com/gorilla/websocket"
)

// Client wraps one WebSocket connection with a buffered outbound queue so a
// slow reader can't block the goroutine broadcasting to other users.
type Client struct {
	conn *websocket.Conn
	send chan []byte
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{conn: conn, send: make(chan []byte, 32)}
}

func (c *Client) Send(payload []byte) {
	select {
	case c.send <- payload:
	default:
		// Outbound buffer full: drop rather than block. The next location
		// update supersedes this one anyway.
	}
}

// WritePump drains the send channel to the socket. Run in its own goroutine.
func (c *Client) WritePump() {
	for payload := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return
		}
	}
}

func (c *Client) Close() {
	close(c.send)
	_ = c.conn.Close()
}

func (c *Client) logReadErr(err error) {
	if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
		log.Printf("websocket read error: %v", err)
	}
}
