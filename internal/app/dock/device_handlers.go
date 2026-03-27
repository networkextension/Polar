package dock

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleDevicePushTokenUpdate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	if userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	var req struct {
		DeviceID   string `json:"device_id"`
		DeviceType string `json:"device_type"`
		PushToken  string `json:"push_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	deviceType := req.DeviceType
	if strings.TrimSpace(deviceType) == "" {
		sessionID, _ := c.Cookie(SessionCookieName)
		if session := s.getSession(sessionID); session != nil {
			deviceType = session.DeviceType
			if strings.TrimSpace(req.DeviceID) == "" {
				req.DeviceID = session.DeviceID
			}
		}
	}

	now := time.Now()
	if err := s.updateUserDevicePushToken(userIDStr, deviceType, req.DeviceID, req.PushToken, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if err := s.syncUserPresence(userIDStr, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Push Token 已更新"})
}

func (s *Server) handleDevicePushTokenDelete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	if userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	var req struct {
		DeviceID string `json:"device_id"`
	}
	_ = c.ShouldBindJSON(&req)
	if strings.TrimSpace(req.DeviceID) == "" {
		sessionID, _ := c.Cookie(SessionCookieName)
		if session := s.getSession(sessionID); session != nil {
			req.DeviceID = session.DeviceID
		}
	}
	if strings.TrimSpace(req.DeviceID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 device_id"})
		return
	}

	now := time.Now()
	if err := s.clearUserDevicePushToken(userIDStr, req.DeviceID, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if err := s.syncUserPresence(userIDStr, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Push Token 已清除"})
}
