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

	cfg := dock.Config{
		Addr:          envOrDefault("ADDR", dock.DefaultAddr),
		PostgresDSN:   envOrDefault("POSTGRES_DSN", dock.DefaultPostgresDSN),
		RedisAddr:     envOrDefault("REDIS_ADDR", dock.DefaultRedisAddr),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       redisDB,
		RedisPrefix:   envOrDefault("REDIS_PREFIX", dock.DefaultRedisPrefix),
		MarkdownDir:   envOrDefault("MARKDOWN_DIR", dock.DefaultMarkdownDir),
		UploadDir:     envOrDefault("UPLOAD_DIR", dock.DefaultUploadDir),
		GeoLiteDBPath: envOrDefault("GEOLITE_DB_PATH", dock.DefaultGeoLiteDBPath),
		PasskeyRPID:   envOrDefault("PASSKEY_RP_ID", dock.DefaultPasskeyRPID),
		PasskeyOrigin: envOrDefault("PASSKEY_ORIGIN", dock.DefaultPasskeyOrigin),
		PasskeyRPName: envOrDefault("PASSKEY_RP_NAME", dock.DefaultPasskeyRPName),
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
