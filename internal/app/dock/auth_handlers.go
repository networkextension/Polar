package dock

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	existingUser, err := s.getUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被注册"})
		return
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	user := &User{
		ID:        generateSessionID()[:16],
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword,
		CreatedAt: time.Now(),
	}
	if err := s.createUser(user); err != nil {
		if err == errEmailExists {
			c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被注册"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	sessionID, err := s.createSession(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.SetCookie(SessionCookieName, sessionID, int(SessionDuration.Seconds()), "/", "", false, true)
	s.recordLoginEvent(c, user.ID, "register")

	c.JSON(http.StatusCreated, gin.H{
		"message":  "注册成功",
		"user_id":  user.ID,
		"username": user.Username,
	})
}

func (s *Server) handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	user, err := s.getUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if user == nil || !checkPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}

	sessionID, err := s.createSession(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.SetCookie(SessionCookieName, sessionID, int(SessionDuration.Seconds()), "/", "", false, true)
	s.recordLoginEvent(c, user.ID, "password")

	c.JSON(http.StatusOK, gin.H{
		"message":  "登录成功",
		"user_id":  user.ID,
		"username": user.Username,
	})
}

func (s *Server) handleLogout(c *gin.Context) {
	sessionID, err := c.Cookie(SessionCookieName)
	if err == nil {
		_ = s.deleteSession(sessionID)
	}

	c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "已成功退出登录"})
}

func (s *Server) handleMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	userIDStr, _ := userID.(string)
	iconURL := ""
	if userIDStr != "" {
		if user, err := s.getUserByID(userIDStr); err == nil && user != nil {
			iconURL = user.IconURL
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"icon_url": iconURL,
	})
}

func (s *Server) handleLoginHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
			return
		}
		limit = parsed
	}

	records, err := s.listLoginRecords(userIDStr, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
	})
}
