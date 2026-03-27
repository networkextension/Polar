package dock

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	apnsHostDev  = "https://api.sandbox.push.apple.com"
	apnsHostProd = "https://api.push.apple.com"
)

type apnsDispatchConfig struct {
	environment string
	host        string
	topic       string
	certPath    string
	authMode    string
	keyID       string
	teamID      string
	p8Path      string
}

type apnsResponse struct {
	Reason string `json:"reason"`
}

type cachedAPNSToken struct {
	Token     string
	ExpiresAt time.Time
}

func (s *Server) runPushDeliveryWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.processPendingPushDeliveries(ctx, 20); err != nil {
				log.Printf("process pending push deliveries failed: %v", err)
			}
		}
	}
}

func (s *Server) processPendingPushDeliveries(ctx context.Context, limit int) error {
	items, err := s.claimPendingPushDeliveries(limit, time.Now())
	if err != nil {
		return err
	}
	for _, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := s.processSinglePushDelivery(item); err != nil {
			log.Printf("push delivery %d failed: %v", item.ID, err)
		}
	}
	return nil
}

func (s *Server) processSinglePushDelivery(item PushDelivery) error {
	now := time.Now()
	log.Printf("push worker: processing delivery id=%d message_id=%d user_id=%s device_id=%s status=%s token=%s",
		item.ID,
		item.MessageID,
		item.UserID,
		item.DeviceID,
		item.Status,
		maskPushToken(item.PushToken),
	)
	if strings.TrimSpace(item.PushToken) == "" {
		log.Printf("push worker: skip delivery id=%d reason=missing_push_token", item.ID)
		return s.updatePushDeliveryResult(item.ID, "failed", "", "missing_push_token", now)
	}

	message, err := s.getChatMessageByID(item.MessageID)
	if err != nil {
		_ = s.updatePushDeliveryResult(item.ID, "failed", "", "load_message_failed:"+err.Error(), now)
		return err
	}
	if message == nil {
		return s.updatePushDeliveryResult(item.ID, "failed", "", "message_not_found", now)
	}

	configs, err := s.availableAPNSConfigs()
	if err != nil {
		_ = s.updatePushDeliveryResult(item.ID, "failed", "", err.Error(), now)
		return err
	}
	if len(configs) == 0 {
		log.Printf("push worker: no APNs config available for delivery id=%d", item.ID)
		return s.updatePushDeliveryResult(item.ID, "failed", "", "apns_not_configured", now)
	}

	payload := s.buildAPNSNotificationPayload(message)
	var lastErr error
	for idx, cfg := range configs {
		log.Printf("push worker: sending delivery id=%d env=%s topic=%s token=%s attempt=%d/%d",
			item.ID,
			cfg.environment,
			cfg.topic,
			maskPushToken(item.PushToken),
			idx+1,
			len(configs),
		)
		apnsID, reason, err := s.sendAPNSNotification(cfg, item.PushToken, payload)
		if err == nil {
			log.Printf("push worker: sent delivery id=%d env=%s apns_id=%s", item.ID, cfg.environment, apnsID)
			return s.updatePushDeliveryResult(item.ID, "sent", apnsID, "", time.Now())
		}
		lastErr = err
		log.Printf("push worker: send failed delivery id=%d env=%s reason=%s err=%v", item.ID, cfg.environment, reason, err)
		if !shouldRetryAlternateAPNSEnvironment(reason) || idx == len(configs)-1 {
			break
		}
		log.Printf("push worker: retry alternate environment for delivery id=%d reason=%s", item.ID, reason)
	}

	errMsg := "apns_send_failed"
	if lastErr != nil {
		errMsg = trimErrorMessage(lastErr.Error(), 512)
	}
	log.Printf("push worker: final failure delivery id=%d error=%s", item.ID, errMsg)
	return s.updatePushDeliveryResult(item.ID, "failed", "", errMsg, time.Now())
}

func (s *Server) buildAPNSNotificationPayload(message *ChatMessage) map[string]any {
	body := "你收到一条新消息"
	var messageID int64
	var threadID int64
	if message != nil {
		messageID = message.ID
		threadID = message.ThreadID
		switch strings.TrimSpace(message.MessageType) {
		case "shared_markdown":
			if strings.TrimSpace(message.MarkdownTitle) != "" {
				body = fmt.Sprintf("%s 分享了 Markdown：%s", message.SenderUsername, message.MarkdownTitle)
			} else {
				body = fmt.Sprintf("%s 分享了一份 Markdown", message.SenderUsername)
			}
		default:
			preview := strings.TrimSpace(message.Content)
			if preview == "" {
				preview = "你收到一条新消息"
			}
			if len([]rune(preview)) > 80 {
				preview = string([]rune(preview)[:80])
			}
			if strings.TrimSpace(message.SenderUsername) != "" {
				body = fmt.Sprintf("%s: %s", message.SenderUsername, preview)
			} else {
				body = preview
			}
		}
	}

	return map[string]any{
		"aps": map[string]any{
			"alert": map[string]any{
				"title": "Polar-",
				"body":  body,
			},
			"sound": "default",
		},
		"message_id": messageID,
		"thread_id":  threadID,
	}
}

func (s *Server) availableAPNSConfigs() ([]apnsDispatchConfig, error) {
	settings, err := s.getSiteSettings()
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, nil
	}

	configs := make([]apnsDispatchConfig, 0, 2)
	if cfg, ok := s.apnsConfigFromCertificate(settings.ApplePushProdCert); ok {
		configs = append(configs, cfg)
	}
	if cfg, ok := s.apnsConfigFromCertificate(settings.ApplePushDevCert); ok {
		configs = append(configs, cfg)
	}
	return configs, nil
}

func (s *Server) apnsConfigFromCertificate(cert *ApplePushCertificate) (apnsDispatchConfig, bool) {
	if cert == nil || strings.TrimSpace(cert.FileURL) == "" {
		return apnsDispatchConfig{}, false
	}
	path, ok := s.applePushCertificatePath(cert)
	if !ok {
		return apnsDispatchConfig{}, false
	}
	switch cert.Environment {
	case "prod":
		topic := firstNonEmpty(s.applePushTopicProd, s.applePushTopic)
		keyID := firstNonEmpty(s.applePushKeyIDProd, s.applePushKeyID)
		teamID := firstNonEmpty(s.applePushTeamIDProd, s.applePushTeamID)
		if strings.HasSuffix(strings.ToLower(path), ".p8") {
			if topic == "" || keyID == "" || teamID == "" {
				return apnsDispatchConfig{}, false
			}
			return apnsDispatchConfig{
				environment: "prod",
				host:        apnsHostProd,
				topic:       topic,
				authMode:    "token",
				keyID:       keyID,
				teamID:      teamID,
				p8Path:      path,
			}, true
		}
		if topic == "" {
			return apnsDispatchConfig{}, false
		}
		return apnsDispatchConfig{
			environment: "prod",
			host:        apnsHostProd,
			topic:       topic,
			authMode:    "certificate",
			certPath:    path,
		}, true
	case "dev":
		topic := firstNonEmpty(s.applePushTopicDev, s.applePushTopic)
		keyID := firstNonEmpty(s.applePushKeyIDDev, s.applePushKeyID)
		teamID := firstNonEmpty(s.applePushTeamIDDev, s.applePushTeamID)
		if strings.HasSuffix(strings.ToLower(path), ".p8") {
			if topic == "" || keyID == "" || teamID == "" {
				return apnsDispatchConfig{}, false
			}
			return apnsDispatchConfig{
				environment: "dev",
				host:        apnsHostDev,
				topic:       topic,
				authMode:    "token",
				keyID:       keyID,
				teamID:      teamID,
				p8Path:      path,
			}, true
		}
		if topic == "" {
			return apnsDispatchConfig{}, false
		}
		return apnsDispatchConfig{
			environment: "dev",
			host:        apnsHostDev,
			topic:       topic,
			authMode:    "certificate",
			certPath:    path,
		}, true
	default:
		return apnsDispatchConfig{}, false
	}
}

func (s *Server) applePushCertificatePath(cert *ApplePushCertificate) (string, bool) {
	if cert == nil || strings.TrimSpace(cert.FileURL) == "" || strings.TrimSpace(s.uploadDir) == "" {
		return "", false
	}
	if !strings.HasPrefix(cert.FileURL, "/uploads/apple_push/") {
		return "", false
	}
	return filepath.Join(s.applePushCertificateDir(), filepath.Base(cert.FileURL)), true
}

func (s *Server) apnsHTTPClient(cfg apnsDispatchConfig) (*http.Client, error) {
	s.apnsMu.Lock()
	defer s.apnsMu.Unlock()

	cacheKey := cfg.environment + "|" + cfg.authMode + "|" + cfg.certPath + "|" + cfg.p8Path
	if client, ok := s.apnsClients[cacheKey]; ok {
		return client, nil
	}

	if cfg.authMode == "token" {
		client := &http.Client{Timeout: 15 * time.Second}
		s.apnsClients[cacheKey] = client
		return client, nil
	}

	if !strings.HasSuffix(strings.ToLower(cfg.certPath), ".pem") {
		return nil, errors.New("unsupported_apns_certificate_format:" + filepath.Ext(cfg.certPath))
	}

	certificate, err := tls.LoadX509KeyPair(cfg.certPath, cfg.certPath)
	if err != nil {
		return nil, fmt.Errorf("load_apns_pem_failed:%w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{certificate},
		},
		ForceAttemptHTTP2: true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}
	s.apnsClients[cacheKey] = client
	return client, nil
}

func (s *Server) sendAPNSNotification(cfg apnsDispatchConfig, pushToken string, payload map[string]any) (string, string, error) {
	client, err := s.apnsHTTPClient(cfg)
	if err != nil {
		return "", "", err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	url := cfg.host + "/3/device/" + strings.TrimSpace(pushToken)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("apns-topic", cfg.topic)
	if cfg.authMode == "token" {
		jwt, err := s.apnsProviderToken(cfg, time.Now())
		if err != nil {
			return "", "", err
		}
		req.Header.Set("authorization", "bearer "+jwt)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	apnsID := strings.TrimSpace(resp.Header.Get("apns-id"))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return apnsID, "", nil
	}

	var parsed apnsResponse
	_ = json.Unmarshal(respBody, &parsed)
	reason := strings.TrimSpace(parsed.Reason)
	if reason == "" {
		reason = strings.TrimSpace(string(respBody))
	}
	if reason == "" {
		reason = resp.Status
	}
	return "", reason, fmt.Errorf("apns_%s_%d:%s", cfg.environment, resp.StatusCode, reason)
}

func shouldRetryAlternateAPNSEnvironment(reason string) bool {
	switch strings.TrimSpace(reason) {
	case "BadDeviceToken", "DeviceTokenNotForTopic":
		return true
	default:
		return false
	}
}

func trimErrorMessage(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}

func maskPushToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 12 {
		return value
	}
	return value[:6] + "..." + value[len(value)-6:]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (s *Server) apnsProviderToken(cfg apnsDispatchConfig, now time.Time) (string, error) {
	cacheKey := strings.Join([]string{
		cfg.environment,
		cfg.keyID,
		cfg.teamID,
		cfg.p8Path,
	}, "|")

	s.apnsMu.Lock()
	cached, ok := s.apnsTokens[cacheKey]
	if ok && strings.TrimSpace(cached.Token) != "" && now.Before(cached.ExpiresAt.Add(-5*time.Minute)) {
		token := cached.Token
		s.apnsMu.Unlock()
		log.Printf("push worker: provider token cache hit env=%s key_id=%s expires_at=%s",
			cfg.environment,
			cfg.keyID,
			cached.ExpiresAt.Format(time.RFC3339),
		)
		return token, nil
	}
	s.apnsMu.Unlock()

	token, err := buildAPNSProviderToken(cfg.keyID, cfg.teamID, cfg.p8Path, now)
	if err != nil {
		return "", err
	}

	s.apnsMu.Lock()
	s.apnsTokens[cacheKey] = cachedAPNSToken{
		Token:     token,
		ExpiresAt: now.Add(1 * time.Hour),
	}
	s.apnsMu.Unlock()
	log.Printf("push worker: provider token refreshed env=%s key_id=%s expires_at=%s",
		cfg.environment,
		cfg.keyID,
		now.Add(1*time.Hour).Format(time.RFC3339),
	)
	return token, nil
}

func buildAPNSProviderToken(keyID, teamID, p8Path string, now time.Time) (string, error) {
	keyID = strings.TrimSpace(keyID)
	teamID = strings.TrimSpace(teamID)
	p8Path = strings.TrimSpace(p8Path)
	if keyID == "" || teamID == "" || p8Path == "" {
		return "", errors.New("apns_token_config_incomplete")
	}

	privateKey, err := loadAPNSPrivateKey(p8Path)
	if err != nil {
		return "", err
	}

	headerJSON, err := json.Marshal(map[string]string{
		"alg": "ES256",
		"kid": keyID,
	})
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(map[string]any{
		"iss": teamID,
		"iat": now.Unix(),
	})
	if err != nil {
		return "", err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	digest := sha256Sum([]byte(unsigned))
	r, sValue, err := ecdsa.Sign(rand.Reader, privateKey, digest)
	if err != nil {
		return "", fmt.Errorf("sign_apns_jwt_failed:%w", err)
	}

	signature := make([]byte, 64)
	copy(signature[32-len(r.Bytes()):32], r.Bytes())
	copy(signature[64-len(sValue.Bytes()):], sValue.Bytes())
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func loadAPNSPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read_apns_p8_failed:%w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("decode_apns_p8_failed")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		ecdsaKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("apns_private_key_not_ecdsa")
		}
		return ecdsaKey, nil
	}
	ecdsaKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse_apns_p8_failed:%w", err)
	}
	return ecdsaKey, nil
}

func sha256Sum(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}
