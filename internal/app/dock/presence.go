package dock

import (
	"log"
	"strings"
	"time"
)

const (
	deviceTypeBrowser = "browser"
	deviceTypeIOS     = "ios"
	deviceTypeAndroid = "android"
)

func normalizeDeviceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case deviceTypeIOS:
		return deviceTypeIOS
	case deviceTypeAndroid:
		return deviceTypeAndroid
	default:
		return deviceTypeBrowser
	}
}

func sanitizePushToken(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 512 {
		return value[:512]
	}
	return value
}

func normalizeDeviceID(value, deviceType string) string {
	value = strings.TrimSpace(value)
	if len(value) > 128 {
		value = value[:128]
	}
	if value != "" {
		return value
	}
	return "default:" + normalizeDeviceType(deviceType)
}

func (s *Server) parseClientInfo(deviceTypeHeader, pushTokenHeader, deviceIDHeader string) (string, string, string) {
	deviceType := normalizeDeviceType(deviceTypeHeader)
	return deviceType, sanitizePushToken(pushTokenHeader), normalizeDeviceID(deviceIDHeader, deviceType)
}

func (s *Server) syncUserPresence(userID string, now time.Time) error {
	_, err := s.db.Exec(
		`UPDATE users u
		    SET is_online = EXISTS(
		            SELECT 1
		              FROM user_devices d
		             WHERE d.user_id = u.id
		               AND d.is_online = TRUE
		        ),
		        last_seen_at = COALESCE(
		            (
		                SELECT MAX(d.last_seen_at)
		                  FROM user_devices d
		                 WHERE d.user_id = u.id
		            ),
		            u.last_seen_at,
		            $2
		        ),
		        last_active_device_type = COALESCE(
		            (
		                SELECT d.device_type
		                  FROM user_devices d
		                 WHERE d.user_id = u.id
		                 ORDER BY d.is_online DESC, d.last_seen_at DESC, d.last_login_at DESC, d.id DESC
		                 LIMIT 1
		            ),
		            u.last_active_device_type,
		            $3
		        )
		  WHERE u.id = $1`,
		userID,
		now,
		deviceTypeBrowser,
	)
	return err
}

func (s *Server) listPresenceSubscribers(userID string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT DISTINCT watcher_id
		   FROM (
		         SELECT CASE WHEN user_low = $1 THEN user_high ELSE user_low END AS watcher_id
		           FROM chat_threads
		          WHERE user_low = $1 OR user_high = $1
		         UNION
		         SELECT $1 AS watcher_id
		        ) peers`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subscribers := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id != "" {
			subscribers = append(subscribers, id)
		}
	}
	return subscribers, rows.Err()
}

func (s *Server) broadcastPresenceUpdate(userID string) {
	if s.wsHub == nil || userID == "" {
		return
	}

	user, err := s.getUserByID(userID)
	if err != nil || user == nil {
		if err != nil {
			log.Printf("load user presence failed: %v", err)
		}
		return
	}

	subscribers, err := s.listPresenceSubscribers(userID)
	if err != nil {
		log.Printf("list presence subscribers failed: %v", err)
		return
	}

	s.broadcastChatEvent(subscribers, chatEvent{
		Type:       "presence",
		UserID:     userID,
		Online:     user.IsOnline,
		DeviceType: user.DeviceType,
		LastSeenAt: user.LastSeenAt,
	})
}

func (s *Server) handlePresenceChange(userID, deviceType, deviceID string, online bool, hasDeviceConnection bool) {
	if userID == "" {
		return
	}

	now := time.Now()
	if err := s.updateUserDevicePresence(userID, deviceType, deviceID, hasDeviceConnection, now); err != nil {
		log.Printf("update user device presence failed: %v", err)
		return
	}
	if err := s.syncUserPresence(userID, now); err != nil {
		log.Printf("sync user presence failed: %v", err)
		return
	}

	s.broadcastPresenceUpdate(userID)
}
