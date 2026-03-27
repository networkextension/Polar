package dock

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsWriteWait  = 10 * time.Second
	wsPongWait   = 60 * time.Second
	wsPingPeriod = (wsPongWait * 9) / 10
	wsMaxMessage = 1024
)

type wsHub struct {
	clients             map[string]map[*wsClient]bool
	threadViewers       map[int64]map[*wsClient]bool
	register            chan *wsClient
	unregister          chan *wsClient
	broadcast           chan wsBroadcast
	updateView          chan wsViewUpdate
	touch               chan *wsClient
	onPresenceChanged   func(userID, deviceType, deviceID string, online bool, deviceOnline bool)
	onThreadViewChanged func(userID, deviceType, deviceID string, threadID int64, viewing bool)
	onConnectionTouched func(userID, deviceType, deviceID, connID string, threadID int64)
}

type wsBroadcast struct {
	userIDs []string
	payload []byte
}

type wsClient struct {
	hub            *wsHub
	server         *Server
	conn           *websocket.Conn
	send           chan []byte
	connID         string
	userID         string
	deviceType     string
	deviceID       string
	activeThreadID int64
}

type wsViewUpdate struct {
	client   *wsClient
	threadID int64
}

func newWSHub() *wsHub {
	return &wsHub{
		clients:       make(map[string]map[*wsClient]bool),
		threadViewers: make(map[int64]map[*wsClient]bool),
		register:      make(chan *wsClient),
		unregister:    make(chan *wsClient),
		broadcast:     make(chan wsBroadcast, 32),
		updateView:    make(chan wsViewUpdate, 32),
		touch:         make(chan *wsClient, 32),
	}
}

func (h *wsHub) run() {
	for {
		select {
		case client := <-h.register:
			group := h.clients[client.userID]
			if group == nil {
				group = make(map[*wsClient]bool)
				h.clients[client.userID] = group
			}
			group[client] = true
			if h.onPresenceChanged != nil {
				go h.onPresenceChanged(client.userID, client.deviceType, client.deviceID, true, true)
			}
			if h.onConnectionTouched != nil {
				go h.onConnectionTouched(client.userID, client.deviceType, client.deviceID, client.connID, client.activeThreadID)
			}
		case client := <-h.unregister:
			h.clearClientThreadView(client)
			group := h.clients[client.userID]
			if group != nil {
				if _, ok := group[client]; ok {
					delete(group, client)
					close(client.send)
				}
				deviceStillOnline := false
				for activeClient := range group {
					if activeClient.deviceType == client.deviceType {
						deviceStillOnline = true
						break
					}
				}
				userStillOnline := len(group) > 0
				if len(group) == 0 {
					delete(h.clients, client.userID)
				}
				if h.onPresenceChanged != nil {
					go h.onPresenceChanged(client.userID, client.deviceType, client.deviceID, userStillOnline, deviceStillOnline)
				}
			}
		case update := <-h.updateView:
			h.setClientThreadView(update.client, update.threadID)
		case client := <-h.touch:
			if h.onConnectionTouched != nil {
				go h.onConnectionTouched(client.userID, client.deviceType, client.deviceID, client.connID, client.activeThreadID)
			}
		case msg := <-h.broadcast:
			for _, userID := range msg.userIDs {
				group := h.clients[userID]
				for client := range group {
					select {
					case client.send <- msg.payload:
					default:
						close(client.send)
						delete(group, client)
					}
				}
			}
		}
	}
}

func (h *wsHub) sendToUsers(userIDs []string, payload []byte) {
	if len(userIDs) == 0 {
		return
	}
	h.broadcast <- wsBroadcast{userIDs: userIDs, payload: payload}
}

func (h *wsHub) setClientThreadView(client *wsClient, threadID int64) {
	if client == nil {
		return
	}
	if client.activeThreadID == threadID {
		return
	}
	h.clearClientThreadView(client)
	if threadID <= 0 {
		return
	}
	viewers := h.threadViewers[threadID]
	if viewers == nil {
		viewers = make(map[*wsClient]bool)
		h.threadViewers[threadID] = viewers
	}
	viewers[client] = true
	client.activeThreadID = threadID
	if h.onThreadViewChanged != nil {
		go h.onThreadViewChanged(client.userID, client.deviceType, client.deviceID, threadID, true)
	}
	if h.onConnectionTouched != nil {
		go h.onConnectionTouched(client.userID, client.deviceType, client.deviceID, client.connID, client.activeThreadID)
	}
}

func (h *wsHub) clearClientThreadView(client *wsClient) {
	if client == nil || client.activeThreadID <= 0 {
		return
	}
	threadID := client.activeThreadID
	viewers := h.threadViewers[threadID]
	if viewers != nil {
		delete(viewers, client)
		if len(viewers) == 0 {
			delete(h.threadViewers, threadID)
		}
	}
	client.activeThreadID = 0
	if h.onThreadViewChanged != nil {
		go h.onThreadViewChanged(client.userID, client.deviceType, client.deviceID, threadID, false)
	}
}

func (c *wsClient) readPump() {
	defer func() {
		if c.server != nil {
			c.server.removeRedisWSConnection(c.userID, c.connID, c.activeThreadID)
		}
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(wsMaxMessage)
	_ = c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	c.conn.SetPongHandler(func(string) error {
		select {
		case c.hub.touch <- c:
		default:
		}
		return c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		select {
		case c.hub.touch <- c:
		default:
		}
		var msg struct {
			Type     string `json:"type"`
			Action   string `json:"action"`
			ThreadID int64  `json:"thread_id"`
		}
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		if msg.Type != "presence" {
			continue
		}
		switch msg.Action {
		case "view_thread":
			c.hub.updateView <- wsViewUpdate{client: c, threadID: msg.ThreadID}
		case "leave_thread":
			c.hub.updateView <- wsViewUpdate{client: c, threadID: 0}
		}
	}
}

func (c *wsClient) writePump() {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
