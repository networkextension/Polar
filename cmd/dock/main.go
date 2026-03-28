package main

import (
	"log"
	"os"
	"strconv"

	"gin-auth-app/internal/app/dock"
)

func main() {
	redisDB := dock.DefaultRedisDB
	if value := os.Getenv("REDIS_DB"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			log.Fatalf("invalid REDIS_DB: %v", err)
		}
		redisDB = parsed
	}

	smtpPort := 587
	if value := os.Getenv("SMTP_PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			log.Fatalf("invalid SMTP_PORT: %v", err)
		}
		smtpPort = parsed
	}

	cfg := dock.Config{
		Addr:                envOrDefault("ADDR", dock.DefaultAddr),
		PostgresDSN:         envOrDefault("POSTGRES_DSN", dock.DefaultPostgresDSN),
		RedisAddr:           envOrDefault("REDIS_ADDR", dock.DefaultRedisAddr),
		RedisPassword:       os.Getenv("REDIS_PASSWORD"),
		RedisDB:             redisDB,
		RedisPrefix:         envOrDefault("REDIS_PREFIX", dock.DefaultRedisPrefix),
		MarkdownDir:         envOrDefault("MARKDOWN_DIR", dock.DefaultMarkdownDir),
		UploadDir:           envOrDefault("UPLOAD_DIR", dock.DefaultUploadDir),
		GeoLiteDBPath:       envOrDefault("GEOLITE_DB_PATH", dock.DefaultGeoLiteDBPath),
		PasskeyRPID:         envOrDefault("PASSKEY_RP_ID", dock.DefaultPasskeyRPID),
		PasskeyOrigin:       envOrDefault("PASSKEY_ORIGIN", dock.DefaultPasskeyOrigin),
		PasskeyRPName:       envOrDefault("PASSKEY_RP_NAME", dock.DefaultPasskeyRPName),
		PublicBaseURL:       envOrDefault("PUBLIC_BASE_URL", envOrDefault("PASSKEY_ORIGIN", dock.DefaultPasskeyOrigin)),
		AIAgentAPIKey:       os.Getenv("AI_AGENT_API_KEY"),
		AIAgentBaseURL:      envOrDefault("AI_AGENT_BASE_URL", "https://api.openai.com/v1/chat/completions"),
		AIAgentModel:        envOrDefault("AI_AGENT_MODEL", "gpt-4.1-mini"),
		AIAgentSystemPrompt: envOrDefault("AI_AGENT_SYSTEM_PROMPT", "你是站内 system 助理。请结合项目运行目录中的文档和用户提问，给出简洁、准确、可执行的中文回答。如果资料不足，请明确说明。"),
		ApplePushTopic:      os.Getenv("APPLE_PUSH_TOPIC"),
		ApplePushTopicDev:   os.Getenv("APPLE_PUSH_TOPIC_DEV"),
		ApplePushTopicProd:  os.Getenv("APPLE_PUSH_TOPIC_PROD"),
		ApplePushKeyID:      os.Getenv("APPLE_PUSH_KEY_ID"),
		ApplePushKeyIDDev:   os.Getenv("APPLE_PUSH_KEY_ID_DEV"),
		ApplePushKeyIDProd:  os.Getenv("APPLE_PUSH_KEY_ID_PROD"),
		ApplePushTeamID:     os.Getenv("APPLE_PUSH_TEAM_ID"),
		ApplePushTeamIDDev:  os.Getenv("APPLE_PUSH_TEAM_ID_DEV"),
		ApplePushTeamIDProd: os.Getenv("APPLE_PUSH_TEAM_ID_PROD"),
		SMTPHost:            os.Getenv("SMTP_HOST"),
		SMTPPort:            smtpPort,
		SMTPUsername:        os.Getenv("SMTP_USERNAME"),
		SMTPPassword:        os.Getenv("SMTP_PASSWORD"),
		SMTPFromEmail:       os.Getenv("SMTP_FROM_EMAIL"),
		SMTPFromName:        os.Getenv("SMTP_FROM_NAME"),

		// Cloudflare R2 — chat attachment storage (optional)
		CloudflareR2AccountID:       os.Getenv("CF_R2_ACCOUNT_ID"),
		CloudflareR2AccessKeyID:     os.Getenv("CF_R2_ACCESS_KEY_ID"),
		CloudflareR2SecretAccessKey: os.Getenv("CF_R2_SECRET_ACCESS_KEY"),
		CloudflareR2Bucket:          os.Getenv("CF_R2_BUCKET"),
		CloudflareR2PublicURL:       os.Getenv("CF_R2_PUBLIC_URL"),
	}

	server, err := dock.NewServer(cfg)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			log.Printf("close db: %v", err)
		}
	}()

	if err := server.Run(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
