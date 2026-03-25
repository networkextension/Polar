package dock

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleLLMConfigList(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	items, err := s.listLLMConfigs(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"configs": items})
}

func (s *Server) handleLLMConfigTest(c *gin.Context) {
	var req struct {
		BaseURL      string `json:"base_url" binding:"required"`
		Model        string `json:"model" binding:"required"`
		APIKey       string `json:"api_key" binding:"required"`
		SystemPrompt string `json:"system_prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}
	if s.aiAgent == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI Agent 未初始化"})
		return
	}

	runtimeConfig := aiRuntimeConfig{
		APIKey:       strings.TrimSpace(req.APIKey),
		BaseURL:      strings.TrimSpace(req.BaseURL),
		Model:        strings.TrimSpace(req.Model),
		SystemPrompt: strings.TrimSpace(req.SystemPrompt),
	}
	if err := s.aiAgent.testRuntimeConfig(runtimeConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "连接成功，模型配置可用"})
}

func (s *Server) handleLLMConfigCreate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	var req struct {
		Name         string `json:"name" binding:"required"`
		BaseURL      string `json:"base_url" binding:"required"`
		Model        string `json:"model" binding:"required"`
		APIKey       string `json:"api_key"`
		SystemPrompt string `json:"system_prompt"`
		Shared       bool   `json:"shared"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}
	name := strings.TrimSpace(req.Name)
	baseURL := strings.TrimSpace(req.BaseURL)
	model := strings.TrimSpace(req.Model)
	if name == "" || baseURL == "" || model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "名称、Base URL 和 Model 不能为空"})
		return
	}
	item, err := s.createLLMConfig(
		userIDStr,
		name,
		baseURL,
		model,
		strings.TrimSpace(req.APIKey),
		strings.TrimSpace(req.SystemPrompt),
		generateSessionID()[:24],
		req.Shared,
		time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"config": item, "message": "配置已创建"})
}

func (s *Server) handleLLMConfigUpdate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效配置"})
		return
	}
	var req struct {
		Name         string `json:"name" binding:"required"`
		BaseURL      string `json:"base_url" binding:"required"`
		Model        string `json:"model" binding:"required"`
		APIKey       string `json:"api_key"`
		SystemPrompt string `json:"system_prompt"`
		Shared       bool   `json:"shared"`
		UpdateAPIKey bool   `json:"update_api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}
	name := strings.TrimSpace(req.Name)
	baseURL := strings.TrimSpace(req.BaseURL)
	model := strings.TrimSpace(req.Model)
	if name == "" || baseURL == "" || model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "名称、Base URL 和 Model 不能为空"})
		return
	}
	item, err := s.updateLLMConfig(
		userIDStr, id,
		name,
		baseURL,
		model,
		strings.TrimSpace(req.APIKey),
		strings.TrimSpace(req.SystemPrompt),
		req.Shared,
		req.UpdateAPIKey,
		time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"config": item, "message": "配置已更新"})
}

func (s *Server) handleLLMConfigDelete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效配置"})
		return
	}
	ok, err := s.deleteLLMConfig(userIDStr, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "配置已删除"})
}

func (s *Server) handleBotUserList(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	items, err := s.listBotUsers(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bots": items})
}

func (s *Server) handleBotUserCreate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	var req struct {
		Name         string `json:"name" binding:"required"`
		Description  string `json:"description"`
		SystemPrompt string `json:"system_prompt"`
		LLMConfigID  int64  `json:"llm_config_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || req.LLMConfigID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bot 名称和配置不能为空"})
		return
	}
	item, err := s.createBotUser(userIDStr, name, strings.TrimSpace(req.Description), strings.TrimSpace(req.SystemPrompt), req.LLMConfigID, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	if item == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "配置不存在或无权限"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"bot": item, "message": "Bot 已创建"})
}

func (s *Server) handleBotUserUpdate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 Bot"})
		return
	}
	var req struct {
		Name         string `json:"name" binding:"required"`
		Description  string `json:"description"`
		SystemPrompt string `json:"system_prompt"`
		LLMConfigID  int64  `json:"llm_config_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || req.LLMConfigID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bot 名称和配置不能为空"})
		return
	}
	item, err := s.updateBotUser(userIDStr, id, name, strings.TrimSpace(req.Description), strings.TrimSpace(req.SystemPrompt), req.LLMConfigID, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bot 或配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bot": item, "message": "Bot 已更新"})
}

func (s *Server) handleBotUserDelete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 Bot"})
		return
	}
	ok, err := s.deleteBotUser(userIDStr, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bot 不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Bot 已删除"})
}

func (s *Server) handleAvailableLLMConfigList(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	items, err := s.listAvailableLLMConfigs(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"configs": items})
}

func (s *Server) handleLLMConfigGetByShareID(c *gin.Context) {
	shareID := c.Param("shareId")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分享标识"})
		return
	}
	item, err := s.getLLMConfigByShareID(shareID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"config": item})
}
