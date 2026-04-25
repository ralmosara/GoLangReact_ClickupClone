package ws

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
	maxMsgSize = 1 << 20
)

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan Event
	rooms  map[string]bool
	userID uuid.UUID
}

type clientMsg struct {
	Action string `json:"action"`
	Room   string `json:"room"`
}

func (c *Client) readPump() {
	defer func() {
		for room := range c.rooms {
			c.hub.leave(room, c)
		}
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg clientMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Action {
		case "join":
			if c.rooms == nil {
				c.rooms = map[string]bool{}
			}
			c.rooms[msg.Room] = true
			c.hub.join(msg.Room, c)
		case "leave":
			delete(c.rooms, msg.Room)
			c.hub.leave(msg.Room, c)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case e, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteJSON(e); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
