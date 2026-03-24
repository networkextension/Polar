package dock

import "time"

const (
	SessionCookieName    = "session_id"
	DefaultRedisAddr     = "localhost:6379"
	DefaultRedisDB       = 0
	DefaultRedisPrefix   = "polar"
	SessionDuration      = 24 * time.Hour
	DefaultMarkdownDir   = "data/markdown"
	DefaultUploadDir     = "data/uploads"
	DefaultGeoLiteDBPath = "data/GeoLite2-City.mmdb"
	DefaultAddr          = ":8080"
	DefaultPostgresDSN   = "postgres://gin_tester:test123456@localhost:5432/gin_auth?sslmode=disable"
	DefaultPasskeyRPID   = "localhost"
	DefaultPasskeyOrigin = "http://localhost:8080"
	DefaultPasskeyRPName = "Gin Auth Demo"
)

type Config struct {
	Addr                string
	PostgresDSN         string
	RedisAddr           string
	RedisPassword       string
	RedisDB             int
	RedisPrefix         string
	MarkdownDir         string
	UploadDir           string
	GeoLiteDBPath       string
	PasskeyRPID         string
	PasskeyOrigin       string
	PasskeyRPName       string
	AIAgentAPIKey       string
	AIAgentBaseURL      string
	AIAgentModel        string
	AIAgentSystemPrompt string
}
