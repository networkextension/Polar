package dock

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleChatList(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	limit := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
			return
		}
		limit = parsed
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		parsed, err := strconv.Atoi(offsetStr)
		if err != nil || parsed < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
			return
		}
		offset = parsed
	}

	chats, hasMore, err := s.listChatThreads(userIDStr, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	nextOffset := offset + len(chats)
	c.JSON(http.StatusOK, gin.H{
		"chats":       chats,
		"has_more":    hasMore,
		"next_offset": nextOffset,
	})
}

func (s *Server) handleChatStart(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}
	targetID := strings.TrimSpace(req.UserID)
	if targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}
	if targetID == userIDStr {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能和自己聊天"})
		return
	}

	otherUser, err := s.getUserByID(targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if otherUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	thread, err := s.ensureChatThread(userIDStr, targetID, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	summary, err := s.getChatSummary(userIDStr, thread.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if summary == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到会话"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chat": summary,
	})
}

func (s *Server) handleChatMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	threadID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || threadID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话"})
		return
	}

	participant, err := s.isChatParticipant(threadID, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if !participant {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该会话"})
		return
	}

	limit := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
			return
		}
		limit = parsed
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		parsed, err := strconv.Atoi(offsetStr)
		if err != nil || parsed < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
			return
		}
		offset = parsed
	}

	messages, hasMore, err := s.listChatMessages(threadID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	if err := s.markChatRead(threadID, userIDStr, time.Now()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if s.wsHub != nil {
		if userLow, userHigh, err := s.getChatParticipants(threadID); err == nil {
			readAt := time.Now()
			s.broadcastChatEvent([]string{userLow, userHigh}, chatEvent{
				Type:   "read",
				ChatID: threadID,
				UserID: userIDStr,
				ReadAt: &readAt,
			})
		}
	}

	nextOffset := offset + len(messages)
	c.JSON(http.StatusOK, gin.H{
		"messages":    messages,
		"has_more":    hasMore,
		"next_offset": nextOffset,
	})
}

func (s *Server) handleChatSend(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	threadID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || threadID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话"})
		return
	}

	participant, err := s.isChatParticipant(threadID, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if !participant {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该会话"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容不能为空"})
		return
	}

	username, _ := c.Get("username")
	senderName, _ := username.(string)
	msgID, err := s.sendChatMessage(threadID, userIDStr, senderName, content, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	otherUserID, err := s.getChatCounterparty(threadID, userIDStr)
	if err != nil {
		log.Printf("load chat counterparty failed: %v", err)
	} else if otherUserID == systemUserID && userIDStr != systemUserID && s.aiAgent != nil {
		s.aiAgent.enqueue(aiAgentTask{
			ThreadID: threadID,
			UserID:   userIDStr,
			Content:  content,
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "发送成功",
		"id":      msgID,
	})
}

func (s *Server) sendChatMessage(threadID int64, senderID, senderName, content string, now time.Time) (int64, error) {
	msgID, err := s.createChatMessage(threadID, senderID, content, now)
	if err != nil {
		return 0, err
	}
	if s.wsHub == nil {
		return msgID, nil
	}

	if senderName == "" {
		if user, lookupErr := s.getUserByID(senderID); lookupErr == nil && user != nil {
			senderName = user.Username
		}
	}

	var senderIcon string
	if user, lookupErr := s.getUserByID(senderID); lookupErr == nil && user != nil {
		senderIcon = user.IconURL
	}

	userLow, userHigh, err := s.getChatParticipants(threadID)
	if err != nil {
		log.Printf("load chat participants failed: %v", err)
		return msgID, nil
	}
	message := &ChatMessage{
		ID:             msgID,
		ThreadID:       threadID,
		SenderID:       senderID,
		SenderUsername: senderName,
		SenderIcon:     senderIcon,
		Content:        content,
		CreatedAt:      now,
	}
	s.broadcastChatEvent([]string{userLow, userHigh}, chatEvent{
		Type:    "message",
		ChatID:  threadID,
		Message: message,
	})
	return msgID, nil
}

func (s *Server) handleSystemAgentStatus(c *gin.Context) {
	if s.aiAgent == nil {
		c.JSON(http.StatusOK, gin.H{
			"user_id":  systemUserID,
			"username": systemUsername,
			"ready":    false,
			"message":  "AI 助理未初始化",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":  systemUserID,
		"username": systemUsername,
		"ready":    strings.TrimSpace(s.aiAgent.apiKey) != "" && strings.TrimSpace(s.aiAgent.model) != "",
		"message":  fmt.Sprintf("system 助理可通过 user_id=%s 发起私聊", systemUserID),
	})
}

func (s *Server) handleChatDelete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	threadID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || threadID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话"})
		return
	}

	messageID, err := strconv.ParseInt(c.Param("messageId"), 10, 64)
	if err != nil || messageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的消息"})
		return
	}

	participant, err := s.isChatParticipant(threadID, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if !participant {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该会话"})
		return
	}

	deleted, err := s.deleteChatMessage(threadID, messageID, userIDStr, time.Now())
	if err != nil {
		if errors.Is(err, errNotMessageOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "只能撤回自己的消息"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "消息不存在或已撤回"})
		return
	}
	if s.wsHub != nil {
		if userLow, userHigh, err := s.getChatParticipants(threadID); err == nil {
			deletedAt := time.Now()
			s.broadcastChatEvent([]string{userLow, userHigh}, chatEvent{
				Type:      "revoke",
				ChatID:    threadID,
				MessageID: messageID,
				DeletedAt: &deletedAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "已撤回"})
}
