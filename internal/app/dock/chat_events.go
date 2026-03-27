package dock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const (
	chatEventMessageCreated = "chat.message.created"
	chatEventRead           = "chat.message.read"
	chatEventRevoked        = "chat.message.revoked"
)

type chatInternalEvent struct {
	Event      string     `json:"event"`
	ChatID     int64      `json:"chat_id"`
	MessageID  int64      `json:"message_id,omitempty"`
	SenderID   string     `json:"sender_id,omitempty"`
	UserID     string     `json:"user_id,omitempty"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	OccurredAt time.Time  `json:"occurred_at"`
}

func (s *Server) prefixedRedisKey(parts ...string) string {
	prefix := DefaultRedisPrefix
	if s.redisPrefix != "" {
		prefix = s.redisPrefix
	}
	key := prefix
	for _, part := range parts {
		key += ":" + part
	}
	return key
}

func (s *Server) chatEventChannel() string {
	return s.prefixedRedisKey("chat", "events")
}

func (s *Server) wsUserKey(userID string) string {
	return s.prefixedRedisKey("ws", "user", userID)
}

func (s *Server) wsConnKey(connID string) string {
	return s.prefixedRedisKey("ws", "conn", connID)
}

func (s *Server) wsThreadViewersKey(threadID int64) string {
	return s.prefixedRedisKey("ws", "thread", fmt.Sprintf("%d", threadID), "viewers")
}

func (s *Server) pushDedupeKey(userID string, messageID int64) string {
	return s.prefixedRedisKey("push", "dedupe", userID, fmt.Sprintf("%d", messageID))
}

func (s *Server) publishChatInternalEvent(event chatInternalEvent) {
	if s.redis == nil {
		return
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("marshal chat event failed: %v", err)
		return
	}
	if err := s.redis.Publish(context.Background(), s.chatEventChannel(), payload).Err(); err != nil {
		log.Printf("publish chat event failed: %v", err)
	}
}

func (s *Server) runChatEventSubscriber(ctx context.Context) {
	if s.redis == nil {
		return
	}
	pubsub := s.redis.Subscribe(ctx, s.chatEventChannel())
	defer func() {
		_ = pubsub.Close()
	}()
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var event chatInternalEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("decode chat event failed: %v", err)
				continue
			}
			s.handleChatInternalEvent(event)
		}
	}
}

func (s *Server) handleChatInternalEvent(event chatInternalEvent) {
	switch event.Event {
	case chatEventMessageCreated:
		s.handleMessageCreatedEvent(event)
	case chatEventRead:
		s.handleReadEvent(event)
	case chatEventRevoked:
		s.handleRevokeEvent(event)
	}
}

func (s *Server) handleMessageCreatedEvent(event chatInternalEvent) {
	if s.wsHub != nil {
		message, err := s.getChatMessageByID(event.MessageID)
		if err != nil {
			log.Printf("load chat message for event failed: %v", err)
		} else if message != nil {
			if userLow, userHigh, err := s.getChatParticipants(event.ChatID); err == nil {
				s.broadcastChatEvent([]string{userLow, userHigh}, chatEvent{
					Type:    "message",
					ChatID:  event.ChatID,
					Message: message,
				})
			}
		}
	}
	if err := s.preparePushDeliveries(event.ChatID, event.MessageID, event.SenderID); err != nil {
		log.Printf("prepare push deliveries failed: %v", err)
	}
}

func (s *Server) handleReadEvent(event chatInternalEvent) {
	if s.wsHub == nil {
		return
	}
	userLow, userHigh, err := s.getChatParticipants(event.ChatID)
	if err != nil {
		log.Printf("load chat participants failed: %v", err)
		return
	}
	s.broadcastChatEvent([]string{userLow, userHigh}, chatEvent{
		Type:   "read",
		ChatID: event.ChatID,
		UserID: event.UserID,
		ReadAt: event.ReadAt,
	})
}

func (s *Server) handleRevokeEvent(event chatInternalEvent) {
	if s.wsHub == nil {
		return
	}
	userLow, userHigh, err := s.getChatParticipants(event.ChatID)
	if err != nil {
		log.Printf("load chat participants failed: %v", err)
		return
	}
	s.broadcastChatEvent([]string{userLow, userHigh}, chatEvent{
		Type:      "revoke",
		ChatID:    event.ChatID,
		MessageID: event.MessageID,
		UserID:    event.UserID,
		DeletedAt: event.DeletedAt,
	})
}

func (s *Server) preparePushDeliveries(threadID, messageID int64, senderID string) error {
	receiverID, err := s.getChatCounterparty(threadID, senderID)
	if err != nil {
		return err
	}
	if receiverID == "" || receiverID == senderID {
		return nil
	}

	ok, err := s.redis.SetNX(context.Background(), s.pushDedupeKey(receiverID, messageID), "1", 24*time.Hour).Result()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	devices, err := s.listPushableUserDevices(receiverID)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return nil
	}

	online, err := s.hasActiveWSConnections(receiverID)
	if err != nil {
		return err
	}
	viewing, err := s.isUserViewingThread(receiverID, threadID)
	if err != nil {
		return err
	}

	now := time.Now()
	status := "pending"
	reason := ""
	if online && viewing {
		status = "skipped"
		reason = "active_in_thread"
	} else if online {
		reason = "online_not_viewing"
	} else {
		reason = "offline"
	}

	for _, device := range devices {
		if err := s.createPushDelivery(messageID, receiverID, device.DeviceID, device.PushToken, status, reason, now); err != nil {
			log.Printf("create push delivery failed: %v", err)
		}
	}
	return nil
}

func (s *Server) syncRedisWSConnection(userID, connID, deviceType, deviceID string, threadID int64) {
	if s.redis == nil || userID == "" || connID == "" {
		return
	}
	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)
	if err := s.redis.SAdd(ctx, s.wsUserKey(userID), connID).Err(); err != nil {
		log.Printf("redis sadd ws user failed: %v", err)
	}
	values := map[string]any{
		"user_id":      userID,
		"device_type":  normalizeDeviceType(deviceType),
		"device_id":    normalizeDeviceID(deviceID, deviceType),
		"thread_id":    fmt.Sprintf("%d", threadID),
		"last_seen_at": now,
	}
	if err := s.redis.HSet(ctx, s.wsConnKey(connID), values).Err(); err != nil {
		log.Printf("redis hset ws conn failed: %v", err)
	}
	if err := s.redis.Expire(ctx, s.wsConnKey(connID), 90*time.Second).Err(); err != nil {
		log.Printf("redis expire ws conn failed: %v", err)
	}
}

func (s *Server) removeRedisWSConnection(userID, connID string, threadID int64) {
	if s.redis == nil || userID == "" || connID == "" {
		return
	}
	ctx := context.Background()
	if err := s.redis.SRem(ctx, s.wsUserKey(userID), connID).Err(); err != nil {
		log.Printf("redis srem ws user failed: %v", err)
	}
	if err := s.redis.Del(ctx, s.wsConnKey(connID)).Err(); err != nil {
		log.Printf("redis del ws conn failed: %v", err)
	}
	if threadID > 0 {
		if err := s.redis.SRem(ctx, s.wsThreadViewersKey(threadID), userID).Err(); err != nil {
			log.Printf("redis srem ws viewers failed: %v", err)
		}
	}
}

func (s *Server) touchRedisWSConnection(userID, connID, deviceType, deviceID string, threadID int64) {
	if s.redis == nil || connID == "" {
		return
	}
	s.syncRedisWSConnection(userID, connID, deviceType, deviceID, threadID)
}

func (s *Server) setRedisThreadViewing(userID string, threadID int64, viewing bool) {
	if s.redis == nil || userID == "" || threadID <= 0 {
		return
	}
	ctx := context.Background()
	key := s.wsThreadViewersKey(threadID)
	var err error
	if viewing {
		err = s.redis.SAdd(ctx, key, userID).Err()
	} else {
		err = s.redis.SRem(ctx, key, userID).Err()
	}
	if err != nil {
		log.Printf("redis thread viewing update failed: %v", err)
	}
}

func (s *Server) hasActiveWSConnections(userID string) (bool, error) {
	if s.redis == nil || userID == "" {
		return false, nil
	}
	count, err := s.redis.SCard(context.Background(), s.wsUserKey(userID)).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Server) isUserViewingThread(userID string, threadID int64) (bool, error) {
	if s.redis == nil || userID == "" || threadID <= 0 {
		return false, nil
	}
	return s.redis.SIsMember(context.Background(), s.wsThreadViewersKey(threadID), userID).Result()
}

func (s *Server) handleThreadViewChange(userID, deviceType, deviceID string, threadID int64, viewing bool) {
	if userID == "" || threadID <= 0 {
		return
	}
	participant, err := s.isChatParticipant(threadID, userID)
	if err != nil {
		log.Printf("check chat participant for thread view failed: %v", err)
		return
	}
	if !participant {
		return
	}
	s.setRedisThreadViewing(userID, threadID, viewing)
	if viewing {
		if err := s.upsertChatMemberStateViewed(threadID, userID, time.Now()); err != nil {
			log.Printf("update chat member state failed: %v", err)
		}
	}
}

func (s *Server) handleConnectionTouch(userID, deviceType, deviceID, connID string, threadID int64) {
	if userID == "" || connID == "" {
		return
	}
	now := time.Now()
	s.touchRedisWSConnection(userID, connID, deviceType, deviceID, threadID)
	if err := s.updateUserDevicePresence(userID, deviceType, deviceID, true, now); err != nil {
		log.Printf("touch user device presence failed: %v", err)
		return
	}
	if err := s.syncUserPresence(userID, now); err != nil {
		log.Printf("touch sync user presence failed: %v", err)
	}
}
