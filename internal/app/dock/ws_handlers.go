package dock

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type chatEvent struct {
	Type       string       `json:"type"`
	ChatID     int64        `json:"chat_id,omitempty"`
	Message    *ChatMessage `json:"message,omitempty"`
	MessageID  int64        `json:"message_id,omitempty"`
	UserID     string       `json:"user_id,omitempty"`
	ReadAt     *time.Time   `json:"read_at,omitempty"`
	DeletedAt  *time.Time   `json:"deleted_at,omitempty"`
	Online     bool         `json:"online,omitempty"`
	DeviceType string       `json:"device_type,omitempty"`
	LastSeenAt *time.Time   `json:"last_seen_at,omitempty"`
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) handleChatWS(c *gin.Context) {
	sessionID, err := c.Cookie(SessionCookieName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	session := s.getSession(sessionID)
	if session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &wsClient{
		hub:        s.wsHub,
		server:     s,
		conn:       conn,
		send:       make(chan []byte, 32),
		connID:     generateSessionID()[:16],
		userID:     session.UserID,
		deviceType: normalizeDeviceType(session.DeviceType),
		deviceID:   normalizeDeviceID(session.DeviceID, session.DeviceType),
	}

	s.wsHub.register <- client

	go client.writePump()
	client.readPump()
}

func (s *Server) broadcastChatEvent(userIDs []string, event chatEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	s.wsHub.sendToUsers(userIDs, payload)
}
