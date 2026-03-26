package dock

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
)

const passkeySessionTTL = 5 * time.Minute

type passkeySession struct {
	Data      webauthn.SessionData
	UserID    string
	Kind      string
	ExpiresAt time.Time
}

func (s *Server) storePasskeySession(kind, userID string, data *webauthn.SessionData) string {
	id := generateSessionID()
	s.passkeyMu.Lock()
	defer s.passkeyMu.Unlock()
	now := time.Now()
	for key, session := range s.passkeySessions {
		if now.After(session.ExpiresAt) {
			delete(s.passkeySessions, key)
		}
	}
	if data == nil {
		data = &webauthn.SessionData{}
	}
	s.passkeySessions[id] = passkeySession{
		Data:      *data,
		UserID:    userID,
		Kind:      kind,
		ExpiresAt: now.Add(passkeySessionTTL),
	}
	return id
}

func (s *Server) consumePasskeySession(id, kind string) (passkeySession, bool) {
	s.passkeyMu.Lock()
	defer s.passkeyMu.Unlock()
	session, ok := s.passkeySessions[id]
	if !ok {
		return passkeySession{}, false
	}
	delete(s.passkeySessions, id)
	if session.Kind != kind || time.Now().After(session.ExpiresAt) {
		return passkeySession{}, false
	}
	return session, true
}

func (s *Server) buildWebAuthnUser(user *User) (*webAuthnUser, error) {
	creds, err := s.listWebAuthnCredentials(user.ID)
	if err != nil {
		return nil, err
	}
	return &webAuthnUser{
		id:          []byte(user.ID),
		name:        user.Email,
		displayName: user.Username,
		credentials: creds,
	}, nil
}

func (s *Server) webAuthnForRequest(c *gin.Context) (*webauthn.WebAuthn, error) {
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		return s.webAuthn, nil
	}

	rpID := hostFromOrigin(origin)
	if rpID == "" {
		rpID = s.passkeyRPID
	}

	return webauthn.New(&webauthn.Config{
		RPDisplayName: s.passkeyRPName,
		RPID:          rpID,
		RPOrigins:     []string{origin},
	})
}

func hostFromOrigin(origin string) string {
	parsed, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	host := parsed.Host
	if host == "" {
		return ""
	}
	if strings.Contains(host, ":") {
		if splitHost, _, err := net.SplitHostPort(host); err == nil && splitHost != "" {
			host = splitHost
		}
	}
	host = strings.Trim(host, "[]")
	return host
}

func (s *Server) handlePasskeyRegisterBegin(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	user, err := s.getUserByID(userIDStr)
	if err != nil || user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	waUser, err := s.buildWebAuthnUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	wa, err := s.webAuthnForRequest(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建 Passkey 失败"})
		return
	}

	options, sessionData, err := wa.BeginRegistration(waUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建 Passkey 失败"})
		return
	}

	sessionID := s.storePasskeySession("register", user.ID, sessionData)
	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"publicKey":  options.Response,
	})
}

func (s *Server) handlePasskeyRegisterFinish(c *gin.Context) {
	sessionID := c.GetHeader("X-Passkey-Session")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少会话信息"})
		return
	}

	session, ok := s.consumePasskeySession(sessionID, "register")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话已过期，请重试"})
		return
	}

	user, err := s.getUserByID(session.UserID)
	if err != nil || user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	waUser, err := s.buildWebAuthnUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	wa, err := s.webAuthnForRequest(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Passkey 校验失败"})
		return
	}

	credential, err := wa.FinishRegistration(waUser, session.Data, c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Passkey 校验失败", "detail": err.Error()})
		return
	}

	if err := s.upsertWebAuthnCredential(user.ID, credential); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存 Passkey 失败"})
		return
	}

	items, err := s.listWebAuthnCredentialSummaries(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Passkey 绑定成功",
		"credentials":  items,
		"count":        len(items),
		"has_passkeys": len(items) > 0,
	})
}

func (s *Server) handlePasskeyList(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	items, err := s.listWebAuthnCredentialSummaries(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"credentials":  items,
		"count":        len(items),
		"has_passkeys": len(items) > 0,
	})
}

func (s *Server) handlePasskeyDelete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	credentialID := strings.TrimSpace(c.Param("credentialId"))
	if credentialID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Passkey"})
		return
	}

	deleted, err := s.deleteWebAuthnCredential(userIDStr, credentialID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除 Passkey 失败"})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Passkey 不存在"})
		return
	}

	items, err := s.listWebAuthnCredentialSummaries(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Passkey 已删除",
		"credentials":  items,
		"count":        len(items),
		"has_passkeys": len(items) > 0,
	})
}

func (s *Server) handlePasskeyLoginBegin(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
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
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户不存在或未绑定 Passkey"})
		return
	}

	waUser, err := s.buildWebAuthnUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if len(waUser.credentials) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户不存在或未绑定 Passkey"})
		return
	}

	wa, err := s.webAuthnForRequest(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建 Passkey 失败"})
		return
	}

	options, sessionData, err := wa.BeginLogin(waUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建 Passkey 失败"})
		return
	}

	sessionID := s.storePasskeySession("login", user.ID, sessionData)
	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"publicKey":  options.Response,
	})
}

func (s *Server) handlePasskeyLoginFinish(c *gin.Context) {
	sessionID := c.GetHeader("X-Passkey-Session")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少会话信息"})
		return
	}

	session, ok := s.consumePasskeySession(sessionID, "login")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话已过期，请重试"})
		return
	}

	user, err := s.getUserByID(session.UserID)
	if err != nil || user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	waUser, err := s.buildWebAuthnUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	wa, err := s.webAuthnForRequest(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Passkey 验证失败"})
		return
	}

	credential, err := wa.FinishLogin(waUser, session.Data, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Passkey 验证失败", "detail": err.Error()})
		return
	}

	if err := s.upsertWebAuthnCredential(user.ID, credential); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新 Passkey 失败"})
		return
	}

	deviceType, pushToken := s.parseLoginClientInfo(c.GetHeader("X-Device-Type"), c.GetHeader("X-Push-Token"))
	now := time.Now()
	if err := s.upsertUserDevice(user.ID, deviceType, pushToken, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if err := s.syncUserPresence(user.ID, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	sessionIDValue, err := s.createSession(user, deviceType, pushToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.SetCookie(SessionCookieName, sessionIDValue, int(SessionDuration.Seconds()), "/", "", false, true)
	s.recordLoginEvent(c, user.ID, "passkey", deviceType)

	c.JSON(http.StatusOK, gin.H{
		"message":  "登录成功",
		"user_id":  user.ID,
		"username": user.Username,
	})
}

type webAuthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte {
	return u.id
}

func (u *webAuthnUser) WebAuthnName() string {
	return u.name
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	return u.displayName
}

func (u *webAuthnUser) WebAuthnIcon() string {
	return ""
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}
