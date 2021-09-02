package server

import (
	"log"
	"time"

	"github.com/evan-buss/openbooks/irc"
	"github.com/google/uuid"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	// Unique ID for the client
	uuid uuid.UUID

	// The websocket connection.
	conn *websocket.Conn

	// Signal to indicate the connection should be terminated.
	disconnect chan struct{}

	// Message to send to the client ws connection
	send chan interface{}

	// Individual IRC connection per connected client.
	irc *irc.Conn
}

func (s *server) closeClient(c *Client) {
	c.irc.Disconnect()
	c.conn.Close()
	s.unregister <- c
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (s *server) readPump(c *Client) {
	defer func() {
		s.closeClient(c)
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		select {
		case <-c.disconnect:
			return
		default:
			var request Request
			err := c.conn.ReadJSON(&request)

			if err != nil {
				log.Printf("%s: Connection Closed: %v", c.uuid.String(), err)
				return
			}

			log.Printf("%s: %s Message Received\n", c.uuid.String(), messageToString(request.RequestType))

			s.routeMessage(request, c)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (s *server) writePump(c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		s.closeClient(c)
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteJSON(message)
			if err != nil {
				log.Printf("%s: Error writing JSON to websocket: %s", c.uuid.String(), err.Error())
				return
			}
		case <-c.disconnect:
			return
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
