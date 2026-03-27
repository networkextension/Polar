package dock

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

	llmThreadID, activeLLMThread, err := s.resolveChatLLMThread(threadID, userIDStr, c.Query("llm_thread_id"), false, time.Now())
	if err != nil {
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

	messages, hasMore, err := s.listChatMessages(threadID, llmThreadID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	var lastReadMessageID *int64
	if len(messages) > 0 {
		lastReadMessageID = &messages[len(messages)-1].ID
	}
	readAt := time.Now()
	if err := s.markChatRead(threadID, userIDStr, readAt, lastReadMessageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if err := s.upsertChatMemberStateViewed(threadID, userIDStr, readAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	s.publishChatInternalEvent(chatInternalEvent{
		Event:  chatEventRead,
		ChatID: threadID,
		UserID: userIDStr,
		ReadAt: &readAt,
	})

	nextOffset := offset + len(messages)
	c.JSON(http.StatusOK, gin.H{
		"messages":      messages,
		"has_more":      hasMore,
		"next_offset":   nextOffset,
		"active_thread": activeLLMThread,
		"active_thread_id": func() any {
			if llmThreadID == nil {
				return nil
			}
			return *llmThreadID
		}(),
	})
}

func (s *Server) handleChatLLMThreads(c *gin.Context) {
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

	_, activeThread, err := s.resolveChatLLMThread(threadID, userIDStr, c.Query("active_thread_id"), true, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	if activeThread == nil {
		c.JSON(http.StatusOK, gin.H{"threads": []LLMThread{}})
		return
	}

	items, err := s.listLLMThreads(threadID, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"threads":       items,
		"active_thread": activeThread,
	})
}

func (s *Server) handleChatLLMThreadCreate(c *gin.Context) {
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
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	botUserID, err := s.getAIResponderForChat(threadID, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if botUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前会话不是 AI Bot 会话"})
		return
	}

	item, err := s.createLLMThread(threadID, userIDStr, botUserID, strings.TrimSpace(req.Title), time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	items, err := s.listLLMThreads(threadID, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"thread":  item,
		"threads": items,
		"message": "新话题已创建",
	})
}

func (s *Server) handleChatLLMThreadUpdate(c *gin.Context) {
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

	llmThreadID, err := strconv.ParseInt(c.Param("threadId"), 10, 64)
	if err != nil || llmThreadID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的话题"})
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	item, err := s.updateLLMThreadTitle(threadID, userIDStr, llmThreadID, req.Title, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "话题不存在"})
		return
	}

	items, err := s.listLLMThreads(threadID, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"thread":  item,
		"threads": items,
		"message": "话题标题已更新",
	})
}

func (s *Server) handleChatLLMThreadDelete(c *gin.Context) {
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

	llmThreadID, err := strconv.ParseInt(c.Param("threadId"), 10, 64)
	if err != nil || llmThreadID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的话题"})
		return
	}

	deleted, err := s.deleteLLMThread(threadID, userIDStr, llmThreadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "话题不存在"})
		return
	}

	items, err := s.listLLMThreads(threadID, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"threads":       items,
		"message":       "话题已删除",
		"thread":        nil,
		"active_thread": nil,
	})
}

func (s *Server) handleChatLLMThreadConfigUpdate(c *gin.Context) {
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

	llmThreadID, err := strconv.ParseInt(c.Param("threadId"), 10, 64)
	if err != nil || llmThreadID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的话题"})
		return
	}

	botUserID, err := s.getAIResponderForChat(threadID, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if botUserID == "" || !strings.HasPrefix(botUserID, "bot_") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前话题不支持切换模型"})
		return
	}

	var req struct {
		LLMConfigID int64 `json:"llm_config_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.LLMConfigID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	item, err := s.updateLLMThreadConfig(threadID, userIDStr, llmThreadID, req.LLMConfigID, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "话题或模型配置不存在"})
		return
	}

	items, err := s.listLLMThreads(threadID, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"thread":  item,
		"threads": items,
		"message": "当前话题模型已切换，后续回复将使用新配置",
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
		Content     string `json:"content" binding:"required"`
		LLMThreadID *int64 `json:"llm_thread_id"`
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

	llmThreadID, activeLLMThread, err := s.resolveChatLLMThread(threadID, userIDStr, func() string {
		if req.LLMThreadID == nil {
			return ""
		}
		return strconv.FormatInt(*req.LLMThreadID, 10)
	}(), true, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	username, _ := c.Get("username")
	senderName, _ := username.(string)
	msgID, err := s.sendChatMessage(threadID, llmThreadID, userIDStr, senderName, content, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	otherUserID, err := s.getChatCounterparty(threadID, userIDStr)
	if err != nil {
		log.Printf("load chat counterparty failed: %v", err)
	} else if userIDStr != otherUserID && s.aiAgent != nil {
		responderName := ""
		switch {
		case otherUserID == systemUserID:
			responderName = systemUsername
		default:
			botUser, botErr := s.getBotUserByUserID(otherUserID)
			if botErr != nil {
				log.Printf("load bot user failed: %v", botErr)
				break
			}
			if botUser != nil {
				responderName = botUser.Name
			}
		}
		if responderName != "" {
			s.aiAgent.enqueue(aiAgentTask{
				ThreadID:        threadID,
				LLMThreadID:     llmThreadID,
				UserID:          userIDStr,
				ResponderUserID: otherUserID,
				ResponderName:   responderName,
				Content:         content,
			})
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "发送成功",
		"id":            msgID,
		"active_thread": activeLLMThread,
	})
}

func (s *Server) handleChatRetry(c *gin.Context) {
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

	messageID, err := strconv.ParseInt(c.Param("messageId"), 10, 64)
	if err != nil || messageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效消息"})
		return
	}

	if s.aiAgent == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI Agent 未就绪"})
		return
	}

	task, sourceContent, err := s.buildRetryTask(threadID, messageID, userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.aiAgent.enqueue(task)
	if deleted, deleteErr := s.markChatMessageFailedResolved(threadID, messageID, time.Now()); deleteErr != nil {
		log.Printf("mark failed retry message resolved failed: %v", deleteErr)
	} else if deleted {
		deletedAt := time.Now()
		s.publishChatInternalEvent(chatInternalEvent{
			Event:     chatEventRevoked,
			ChatID:    threadID,
			MessageID: messageID,
			UserID:    "retry",
			DeletedAt: &deletedAt,
		})
	}
	c.JSON(http.StatusAccepted, gin.H{
		"message": "已重新提交上一条用户消息",
		"content": sourceContent,
	})
}

func (s *Server) buildRetryTask(threadID, messageID int64, requesterID string) (aiAgentTask, string, error) {
	targetMessage, err := s.getChatMessageByID(messageID)
	if err != nil {
		return aiAgentTask{}, "", err
	}
	if targetMessage == nil || targetMessage.ThreadID != threadID {
		return aiAgentTask{}, "", errors.New("消息不存在")
	}

	responderUserID, err := s.getAIResponderForChat(threadID, requesterID)
	if err != nil {
		return aiAgentTask{}, "", err
	}
	if responderUserID == "" {
		return aiAgentTask{}, "", errors.New("当前会话不是 AI Bot 会话")
	}
	if targetMessage.SenderID != responderUserID {
		return aiAgentTask{}, "", errors.New("只能重试 AI 返回的消息")
	}
	if !targetMessage.Failed {
		return aiAgentTask{}, "", errors.New("当前消息不是失败消息")
	}

	sourceMessage, err := s.findRetrySourceMessage(threadID, targetMessage)
	if err != nil {
		return aiAgentTask{}, "", err
	}
	if sourceMessage == nil || strings.TrimSpace(sourceMessage.Content) == "" {
		return aiAgentTask{}, "", errors.New("未找到可重试的上一条用户消息")
	}

	responderName := targetMessage.SenderUsername
	if strings.TrimSpace(responderName) == "" {
		responderName = responderUserID
	}

	return aiAgentTask{
		ThreadID:        threadID,
		LLMThreadID:     targetMessage.LLMThreadID,
		UserID:          requesterID,
		ResponderUserID: responderUserID,
		ResponderName:   responderName,
		Content:         strings.TrimSpace(sourceMessage.Content),
	}, strings.TrimSpace(sourceMessage.Content), nil
}

func (s *Server) getAIResponderForChat(threadID int64, userID string) (string, error) {
	otherUserID, err := s.getChatCounterparty(threadID, userID)
	if err != nil {
		return "", err
	}
	if otherUserID == systemUserID {
		return otherUserID, nil
	}
	botUser, err := s.getBotUserByUserID(otherUserID)
	if err != nil {
		return "", err
	}
	if botUser != nil {
		return otherUserID, nil
	}
	return "", nil
}

func (s *Server) resolveChatLLMThread(threadID int64, userID, requestedThreadID string, autoCreate bool, now time.Time) (*int64, *LLMThread, error) {
	botUserID, err := s.getAIResponderForChat(threadID, userID)
	if err != nil {
		return nil, nil, err
	}
	if botUserID == "" {
		return nil, nil, nil
	}
	if strings.TrimSpace(requestedThreadID) != "" {
		id, parseErr := strconv.ParseInt(strings.TrimSpace(requestedThreadID), 10, 64)
		if parseErr != nil || id <= 0 {
			return nil, nil, nil
		}
		thread, getErr := s.getLLMThread(threadID, userID, id)
		if getErr != nil {
			return nil, nil, getErr
		}
		if thread != nil {
			return &thread.ID, thread, nil
		}
	}
	if !autoCreate {
		return nil, nil, nil
	}
	thread, err := s.ensureDefaultLLMThread(threadID, userID, botUserID, now)
	if err != nil {
		return nil, nil, err
	}
	if thread == nil {
		return nil, nil, nil
	}
	return &thread.ID, thread, nil
}

func (s *Server) sendChatMessage(threadID int64, llmThreadID *int64, senderID, senderName, content string, now time.Time) (int64, error) {
	msgID, err := s.createChatMessage(threadID, llmThreadID, senderID, content, now)
	if err != nil {
		return 0, err
	}
	return s.broadcastChatMessageByID(threadID, msgID, senderID, senderName)
}

func (s *Server) sendFailedBotMessage(threadID int64, llmThreadID *int64, senderID, senderName, content string, now time.Time) (int64, error) {
	msgID, err := s.createChatMessageWithOptions(threadID, llmThreadID, senderID, "text", true, content, nil, "", now)
	if err != nil {
		return 0, err
	}
	return s.broadcastChatMessageByID(threadID, msgID, senderID, senderName)
}

func (s *Server) sendSharedMarkdownMessage(threadID int64, llmThreadID *int64, senderID, senderName string, markdownEntryID int64, markdownTitle, preview string, now time.Time) (int64, error) {
	msgID, err := s.createChatMessageWithMetadata(threadID, llmThreadID, senderID, "shared_markdown", preview, &markdownEntryID, markdownTitle, now)
	if err != nil {
		return 0, err
	}
	return s.broadcastChatMessageByID(threadID, msgID, senderID, senderName)
}

func (s *Server) broadcastChatMessageByID(threadID, messageID int64, senderID, senderName string) (int64, error) {
	_ = senderName
	s.publishChatInternalEvent(chatInternalEvent{
		Event:     chatEventMessageCreated,
		ChatID:    threadID,
		MessageID: messageID,
		SenderID:  senderID,
	})
	return messageID, nil
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

func (s *Server) handleChatSharedMarkdown(c *gin.Context) {
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

	messageID, err := strconv.ParseInt(c.Param("messageId"), 10, 64)
	if err != nil || messageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的消息"})
		return
	}

	message, err := s.getChatMessageByID(messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if message == nil || message.ThreadID != threadID || message.MessageType != "shared_markdown" || message.MarkdownEntryID == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到共享 Markdown"})
		return
	}

	entry, err := s.getMarkdownEntryByID(*message.MarkdownEntryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文档不存在"})
		return
	}

	content, err := os.ReadFile(entry.FilePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entry":    entry,
		"content":  string(content),
		"message":  message,
		"can_edit": false,
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
		deletedAt := time.Now()
		s.publishChatInternalEvent(chatInternalEvent{
			Event:     chatEventRevoked,
			ChatID:    threadID,
			MessageID: messageID,
			UserID:    userIDStr,
			DeletedAt: &deletedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "已撤回"})
}
