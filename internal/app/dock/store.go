package dock

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var errEmailExists = errors.New("email already exists")

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // password_hash
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID        string
	UserID    string
	Username  string
	ExpiresAt time.Time
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	schema := `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	username TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
`
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// 生成随机 Session ID
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (s *Server) createSession(user *User) (string, error) {
	sessionID := generateSessionID()
	session := &Session{
		ID:        sessionID,
		UserID:    user.ID,
		Username:  user.Username,
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	_, err := s.db.Exec(
		`INSERT INTO sessions (id, user_id, username, expires_at) VALUES ($1, $2, $3, $4)`,
		session.ID,
		session.UserID,
		session.Username,
		session.ExpiresAt,
	)
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func (s *Server) getSession(sessionID string) *Session {
	var session Session
	err := s.db.QueryRow(
		`SELECT id, user_id, username, expires_at FROM sessions WHERE id = $1`,
		sessionID,
	).Scan(&session.ID, &session.UserID, &session.Username, &session.ExpiresAt)
	if err != nil {
		return nil
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.deleteSession(sessionID)
		return nil
	}
	return &session
}

func (s *Server) deleteSession(sessionID string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}

func (s *Server) cleanupSessions() {
	for {
		time.Sleep(1 * time.Hour)
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE expires_at < NOW()`)
	}
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *Server) getUserByEmail(email string) (*User, error) {
	var user User
	err := s.db.QueryRow(
		`SELECT id, username, email, password_hash, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (s *Server) createUser(user *User) error {
	_, err := s.db.Exec(
		`INSERT INTO users (id, username, email, password_hash, created_at) VALUES ($1, $2, $3, $4, $5)`,
		user.ID,
		user.Username,
		user.Email,
		user.Password,
		user.CreatedAt,
	)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return errEmailExists
		}
		return err
	}
	return nil
}
