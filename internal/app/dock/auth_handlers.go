package dock

import (
	"log"
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
		jsonError(c, http.StatusBadRequest, "common.invalid_input")
		return
	}

	existingUser, err := s.getUserByEmail(req.Email)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}
	if existingUser != nil {
		jsonError(c, http.StatusConflict, "auth.email_registered")
		return
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	deviceType, pushToken, deviceID := s.parseClientInfo(c.GetHeader("X-Device-Type"), c.GetHeader("X-Push-Token"), c.GetHeader("X-Device-Id"))
	now := time.Now()

	user := &User{
		ID:         generateSessionID()[:16],
		Username:   req.Username,
		Email:      req.Email,
		Password:   hashedPassword,
		DeviceType: deviceType,
		CreatedAt:  now,
	}
	if err := s.createUser(user); err != nil {
		if err == errEmailExists {
			jsonError(c, http.StatusConflict, "auth.email_registered")
			return
		}
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	if err := s.upsertUserDeviceWithID(user.ID, deviceType, deviceID, pushToken, now); err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}
	if err := s.syncUserPresence(user.ID, now); err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	sessionID, err := s.createSession(user, deviceType, deviceID, pushToken)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}
	c.SetCookie(SessionCookieName, sessionID, int(SessionDuration.Seconds()), "/", "", false, true)
	s.recordLoginEvent(c, user.ID, "register", deviceType)

	jsonMessage(c, http.StatusCreated, "auth.register_success", gin.H{
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
		jsonError(c, http.StatusBadRequest, "common.invalid_input")
		return
	}

	user, err := s.getUserByEmail(req.Email)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}
	if user == nil || !checkPassword(req.Password, user.Password) {
		jsonError(c, http.StatusUnauthorized, "auth.invalid_credentials")
		return
	}

	deviceType, pushToken, deviceID := s.parseClientInfo(c.GetHeader("X-Device-Type"), c.GetHeader("X-Push-Token"), c.GetHeader("X-Device-Id"))
	now := time.Now()
	if err := s.upsertUserDeviceWithID(user.ID, deviceType, deviceID, pushToken, now); err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}
	if err := s.syncUserPresence(user.ID, now); err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	sessionID, err := s.createSession(user, deviceType, deviceID, pushToken)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}
	c.SetCookie(SessionCookieName, sessionID, int(SessionDuration.Seconds()), "/", "", false, true)
	s.recordLoginEvent(c, user.ID, "password", deviceType)

	jsonMessage(c, http.StatusOK, "auth.login_success", gin.H{
		"user_id":  user.ID,
		"username": user.Username,
	})
}

func (s *Server) handleLogout(c *gin.Context) {
	sessionID, err := c.Cookie(SessionCookieName)
	if err == nil {
		if err := s.deleteSession(sessionID); err != nil {
			log.Printf("logout delete session failed: %v", err)
		}
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})
	jsonMessage(c, http.StatusOK, "auth.logout_success", nil)
}

func (s *Server) handleMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	userIDStr, _ := userID.(string)
	var user *User
	if userIDStr != "" {
		user, _ = s.getUserByID(userIDStr)
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"email": func() string {
			if user != nil {
				return user.Email
			}
			return ""
		}(),
		"email_verified": func() bool {
			return user != nil && user.EmailVerified
		}(),
		"icon_url": func() string {
			if user != nil {
				return user.IconURL
			}
			return ""
		}(),
		"is_online": func() bool {
			return user != nil && user.IsOnline
		}(),
		"device_type": func() string {
			if user != nil && user.DeviceType != "" {
				return user.DeviceType
			}
			return deviceTypeBrowser
		}(),
		"last_seen_at": func() *time.Time {
			if user != nil {
				return user.LastSeenAt
			}
			return nil
		}(),
		"bio": func() string {
			if user != nil {
				return user.Bio
			}
			return ""
		}(),
	})
}

func (s *Server) handleLoginHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed <= 0 {
			jsonError(c, http.StatusBadRequest, "common.invalid_input")
			return
		}
		limit = parsed
	}

	records, err := s.listLoginRecords(userIDStr, limit)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
	})
}
