package main

import (
	"log"
	"os"

	"gin-auth-app/internal/app/dock"
)

func main() {
	cfg := dock.Config{
		Addr:        envOrDefault("ADDR", dock.DefaultAddr),
		PostgresDSN: envOrDefault("POSTGRES_DSN", dock.DefaultPostgresDSN),
		MarkdownDir: envOrDefault("MARKDOWN_DIR", dock.DefaultMarkdownDir),
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
