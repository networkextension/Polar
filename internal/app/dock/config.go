package dock

import "time"

const (
	SessionCookieName  = "session_id"
	SessionDuration    = 24 * time.Hour
	DefaultMarkdownDir = "data/markdown"
	DefaultAddr        = ":8080"
	DefaultPostgresDSN = "postgres://gin_tester:test123456@localhost:5432/gin_auth?sslmode=disable"
)

type Config struct {
	Addr        string
	PostgresDSN string
	MarkdownDir string
}
