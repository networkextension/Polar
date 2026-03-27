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
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	user, err := s.getUserByID(userIDStr)
	if err != nil || user == nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	waUser, err := s.buildWebAuthnUser(user)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	wa, err := s.webAuthnForRequest(c)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "passkey.create_failed")
		return
	}

	options, sessionData, err := wa.BeginRegistration(waUser)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "passkey.create_failed")
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
		jsonError(c, http.StatusBadRequest, "passkey.session_missing")
		return
	}

	session, ok := s.consumePasskeySession(sessionID, "register")
	if !ok {
		jsonError(c, http.StatusBadRequest, "passkey.session_expired")
		return
	}

	user, err := s.getUserByID(session.UserID)
	if err != nil || user == nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	waUser, err := s.buildWebAuthnUser(user)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	wa, err := s.webAuthnForRequest(c)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "passkey.verify_failed")
		return
	}

	credential, err := wa.FinishRegistration(waUser, session.Data, c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": tr(c, "passkey.verify_failed"), "detail": err.Error()})
		return
	}

	if err := s.upsertWebAuthnCredential(user.ID, credential); err != nil {
		jsonError(c, http.StatusInternalServerError, "passkey.save_failed")
		return
	}

	items, err := s.listWebAuthnCredentialSummaries(user.ID)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	jsonMessage(c, http.StatusOK, "passkey.bound_success", gin.H{
		"credentials":  items,
		"count":        len(items),
		"has_passkeys": len(items) > 0,
	})
}

func (s *Server) handlePasskeyList(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	items, err := s.listWebAuthnCredentialSummaries(userIDStr)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
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
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	credentialID := strings.TrimSpace(c.Param("credentialId"))
	if credentialID == "" {
		jsonError(c, http.StatusBadRequest, "passkey.invalid_credential")
		return
	}

	deleted, err := s.deleteWebAuthnCredential(userIDStr, credentialID)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "passkey.delete_failed")
		return
	}
	if !deleted {
		jsonError(c, http.StatusNotFound, "passkey.not_found")
		return
	}

	items, err := s.listWebAuthnCredentialSummaries(userIDStr)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	jsonMessage(c, http.StatusOK, "passkey.deleted_success", gin.H{
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
		jsonError(c, http.StatusBadRequest, "common.invalid_input")
		return
	}

	user, err := s.getUserByEmail(req.Email)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}
	if user == nil {
		jsonError(c, http.StatusBadRequest, "passkey.user_missing_or_unbound")
		return
	}

	waUser, err := s.buildWebAuthnUser(user)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}
	if len(waUser.credentials) == 0 {
		jsonError(c, http.StatusBadRequest, "passkey.user_missing_or_unbound")
		return
	}

	wa, err := s.webAuthnForRequest(c)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "passkey.create_failed")
		return
	}

	options, sessionData, err := wa.BeginLogin(waUser)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "passkey.create_failed")
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
		jsonError(c, http.StatusBadRequest, "passkey.session_missing")
		return
	}

	session, ok := s.consumePasskeySession(sessionID, "login")
	if !ok {
		jsonError(c, http.StatusBadRequest, "passkey.session_expired")
		return
	}

	user, err := s.getUserByID(session.UserID)
	if err != nil || user == nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	waUser, err := s.buildWebAuthnUser(user)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}

	wa, err := s.webAuthnForRequest(c)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "passkey.verify_failed")
		return
	}

	credential, err := wa.FinishLogin(waUser, session.Data, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": tr(c, "passkey.verify_failed"), "detail": err.Error()})
		return
	}

	if err := s.upsertWebAuthnCredential(user.ID, credential); err != nil {
		jsonError(c, http.StatusInternalServerError, "passkey.update_failed")
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

	sessionIDValue, err := s.createSession(user, deviceType, deviceID, pushToken)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, "common.server_error")
		return
	}
	c.SetCookie(SessionCookieName, sessionIDValue, int(SessionDuration.Seconds()), "/", "", false, true)
	s.recordLoginEvent(c, user.ID, "passkey", deviceType)

	jsonMessage(c, http.StatusOK, "auth.login_success", gin.H{
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
