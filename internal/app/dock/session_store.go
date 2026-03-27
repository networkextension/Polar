package dock

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (s *Server) sessionKey(sessionID string) string {
	prefix := DefaultRedisPrefix
	if s.redisPrefix != "" {
		prefix = s.redisPrefix
	}
	return fmt.Sprintf("%s:session:%s", prefix, sessionID)
}

func (s *Server) createSession(user *User, deviceType, deviceID, pushToken string) (string, error) {
	sessionID := generateSessionID()
	session := &Session{
		ID:         sessionID,
		UserID:     user.ID,
		Username:   user.Username,
		Role:       user.Role,
		DeviceType: normalizeDeviceType(deviceType),
		DeviceID:   normalizeDeviceID(deviceID, deviceType),
		PushToken:  sanitizePushToken(pushToken),
		ExpiresAt:  time.Now().Add(SessionDuration),
	}

	payload, err := json.Marshal(session)
	if err != nil {
		return "", err
	}

	err = s.redis.Set(context.Background(), s.sessionKey(sessionID), payload, SessionDuration).Err()
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func (s *Server) getSession(sessionID string) *Session {
	payload, err := s.redis.Get(context.Background(), s.sessionKey(sessionID)).Bytes()
	if err != nil {
		return nil
	}

	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		_ = s.deleteSession(sessionID)
		return nil
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.deleteSession(sessionID)
		return nil
	}

	return &session
}

func (s *Server) deleteSession(sessionID string) error {
	return s.redis.Del(context.Background(), s.sessionKey(sessionID)).Err()
}
