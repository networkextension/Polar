package dock

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var (
	errEmailExists     = errors.New("email already exists")
	errNotMessageOwner = errors.New("not message owner")
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // password_hash
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID        string
	UserID    string
	Username  string
	Role      string
	ExpiresAt time.Time
}

type MarkdownEntry struct {
	ID         int64     `json:"id"`
	UserID     string    `json:"user_id"`
	Title      string    `json:"title"`
	FilePath   string    `json:"file_path"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type LoginRecord struct {
	ID          int64     `json:"id"`
	UserID      string    `json:"user_id"`
	IPAddress   string    `json:"ip_address"`
	Country     string    `json:"country"`
	Region      string    `json:"region"`
	City        string    `json:"city"`
	LoginMethod string    `json:"login_method"`
	LoggedInAt  time.Time `json:"logged_in_at"`
}

type Tag struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Post struct {
	ID         int64     `json:"id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	TagID      *int64    `json:"tag_id,omitempty"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	LikeCount  int       `json:"like_count"`
	ReplyCount int       `json:"reply_count"`
	LikedByMe  bool      `json:"liked_by_me"`
	Images     []string  `json:"images"`
}

type PostReply struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatThread struct {
	ID            int64      `json:"id"`
	UserLow       string     `json:"user_low"`
	UserHigh      string     `json:"user_high"`
	CreatedAt     time.Time  `json:"created_at"`
	LastMessage   string     `json:"last_message"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
}

type ChatSummary struct {
	ID            int64      `json:"id"`
	OtherUserID   string     `json:"other_user_id"`
	OtherUsername string     `json:"other_username"`
	LastMessage   string     `json:"last_message"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UnreadCount   int        `json:"unread_count"`
}

type ChatMessage struct {
	ID             int64      `json:"id"`
	ThreadID       int64      `json:"thread_id"`
	SenderID       string     `json:"sender_id"`
	SenderUsername string     `json:"sender_username"`
	Content        string     `json:"content"`
	CreatedAt      time.Time  `json:"created_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	DeletedBy      string     `json:"deleted_by,omitempty"`
	Deleted        bool       `json:"deleted"`
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
	role TEXT NOT NULL DEFAULT 'user',
	created_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE users
	ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';

CREATE TABLE IF NOT EXISTS markdown_entries (
	id BIGSERIAL PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	file_path TEXT NOT NULL,
	uploaded_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS webauthn_credentials (
	credential_id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	credential_json JSONB NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS login_records (
	id BIGSERIAL PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	ip_address TEXT NOT NULL,
	country TEXT NOT NULL DEFAULT '',
	region TEXT NOT NULL DEFAULT '',
	city TEXT NOT NULL DEFAULT '',
	login_method TEXT NOT NULL DEFAULT 'password',
	logged_in_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
	id BIGSERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	slug TEXT NOT NULL UNIQUE,
	description TEXT NOT NULL DEFAULT '',
	sort_order INT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS posts (
	id BIGSERIAL PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	tag_id BIGINT REFERENCES tags(id) ON DELETE SET NULL,
	content TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS post_images (
	id BIGSERIAL PRIMARY KEY,
	post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
	file_url TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS post_likes (
	post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (post_id, user_id)
);

CREATE TABLE IF NOT EXISTS post_replies (
	id BIGSERIAL PRIMARY KEY,
	post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	content TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS chat_threads (
	id BIGSERIAL PRIMARY KEY,
	user_low TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	user_high TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL,
	last_message TEXT NOT NULL DEFAULT '',
	last_message_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS chat_messages (
	id BIGSERIAL PRIMARY KEY,
	thread_id BIGINT NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
	sender_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	content TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE chat_messages
	ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE chat_messages
	ADD COLUMN IF NOT EXISTS deleted_by TEXT;

CREATE TABLE IF NOT EXISTS chat_reads (
	thread_id BIGINT NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	last_read_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (thread_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_markdown_entries_user_id ON markdown_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON webauthn_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_login_records_user_id_logged_in_at ON login_records(user_id, logged_in_at DESC);
CREATE INDEX IF NOT EXISTS idx_tags_sort_order_created_at ON tags(sort_order DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_post_images_post_id ON post_images(post_id);
CREATE INDEX IF NOT EXISTS idx_post_likes_post_id ON post_likes(post_id);
CREATE INDEX IF NOT EXISTS idx_post_replies_post_id ON post_replies(post_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_threads_pair ON chat_threads(user_low, user_high);
CREATE INDEX IF NOT EXISTS idx_chat_threads_last_message_at ON chat_threads(last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_thread_id_created_at ON chat_messages(thread_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_reads_user_id ON chat_reads(user_id);
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

func (s *Server) createLoginRecord(record *LoginRecord) error {
	if record == nil {
		return errors.New("login record is nil")
	}
	_, err := s.db.Exec(
		`INSERT INTO login_records (user_id, ip_address, country, region, city, login_method, logged_in_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		record.UserID,
		record.IPAddress,
		record.Country,
		record.Region,
		record.City,
		record.LoginMethod,
		record.LoggedInAt,
	)
	return err
}

func (s *Server) listLoginRecords(userID string, limit int) ([]LoginRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(
		`SELECT id, user_id, ip_address, country, region, city, login_method, logged_in_at
		 FROM login_records
		 WHERE user_id = $1
		 ORDER BY logged_in_at DESC
		 LIMIT $2`,
		userID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]LoginRecord, 0, limit)
	for rows.Next() {
		var record LoginRecord
		if err := rows.Scan(
			&record.ID,
			&record.UserID,
			&record.IPAddress,
			&record.Country,
			&record.Region,
			&record.City,
			&record.LoginMethod,
			&record.LoggedInAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
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
		`SELECT id, username, email, password_hash, role, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (s *Server) getUserByID(userID string) (*User, error) {
	var user User
	err := s.db.QueryRow(
		`SELECT id, username, email, password_hash, role, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (s *Server) createUser(user *User) error {
	if user.Role == "" {
		user.Role = "user"
	}
	_, err := s.db.Exec(
		`INSERT INTO users (id, username, email, password_hash, role, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID,
		user.Username,
		user.Email,
		user.Password,
		user.Role,
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

func (s *Server) createMarkdownEntry(userID, title, filePath string, uploadedAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO markdown_entries (user_id, title, file_path, uploaded_at) VALUES ($1, $2, $3, $4)`,
		userID,
		title,
		filePath,
		uploadedAt,
	)
	return err
}

func (s *Server) createMarkdownEntryReturningID(userID, title, filePath string, uploadedAt time.Time) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO markdown_entries (user_id, title, file_path, uploaded_at) VALUES ($1, $2, $3, $4) RETURNING id`,
		userID,
		title,
		filePath,
		uploadedAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Server) listWebAuthnCredentials(userID string) ([]webauthn.Credential, error) {
	rows, err := s.db.Query(
		`SELECT credential_json FROM webauthn_credentials WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []webauthn.Credential
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var cred webauthn.Credential
		if err := json.Unmarshal(payload, &cred); err != nil {
			return nil, err
		}
		creds = append(creds, cred)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return creds, nil
}

func (s *Server) upsertWebAuthnCredential(userID string, credential *webauthn.Credential) error {
	if credential == nil {
		return errors.New("credential is nil")
	}
	payload, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	_, err = s.db.Exec(
		`INSERT INTO webauthn_credentials (credential_id, user_id, credential_json, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 ON CONFLICT (credential_id)
		 DO UPDATE SET credential_json = EXCLUDED.credential_json, updated_at = NOW()`,
		credentialID,
		userID,
		payload,
	)
	return err
}

func (s *Server) listMarkdownEntries(userID string, limit, offset int) ([]MarkdownEntry, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(
		`SELECT id, user_id, title, file_path, uploaded_at
		FROM markdown_entries
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
		LIMIT $2 OFFSET $3`,
		userID,
		limit+1,
		offset,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	entries := make([]MarkdownEntry, 0, limit+1)
	for rows.Next() {
		var entry MarkdownEntry
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Title, &entry.FilePath, &entry.UploadedAt); err != nil {
			return nil, false, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := false
	if len(entries) > limit {
		hasMore = true
		entries = entries[:limit]
	}
	return entries, hasMore, nil
}

func (s *Server) getMarkdownEntry(userID string, id int64) (*MarkdownEntry, error) {
	var entry MarkdownEntry
	err := s.db.QueryRow(
		`SELECT id, user_id, title, file_path, uploaded_at
		FROM markdown_entries
		WHERE user_id = $1 AND id = $2`,
		userID,
		id,
	).Scan(&entry.ID, &entry.UserID, &entry.Title, &entry.FilePath, &entry.UploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

func (s *Server) updateMarkdownEntry(userID string, id int64, title, filePath string) error {
	_, err := s.db.Exec(
		`UPDATE markdown_entries SET title = $1, file_path = $2 WHERE user_id = $3 AND id = $4`,
		title,
		filePath,
		userID,
		id,
	)
	return err
}

func (s *Server) deleteMarkdownEntry(userID string, id int64) error {
	_, err := s.db.Exec(
		`DELETE FROM markdown_entries WHERE user_id = $1 AND id = $2`,
		userID,
		id,
	)
	return err
}

func (s *Server) createTag(tag *Tag) (*Tag, error) {
	if tag == nil {
		return nil, errors.New("tag is nil")
	}
	now := time.Now()
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = now
	}
	if tag.UpdatedAt.IsZero() {
		tag.UpdatedAt = now
	}
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO tags (name, slug, description, sort_order, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		tag.Name,
		tag.Slug,
		tag.Description,
		tag.SortOrder,
		tag.CreatedAt,
		tag.UpdatedAt,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	tag.ID = id
	return tag, nil
}

func (s *Server) listTags(limit, offset int) ([]Tag, bool, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(
		`SELECT id, name, slug, description, sort_order, created_at, updated_at
		 FROM tags
		 ORDER BY sort_order DESC, created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit+1,
		offset,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	tags := make([]Tag, 0, limit+1)
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.Slug,
			&tag.Description,
			&tag.SortOrder,
			&tag.CreatedAt,
			&tag.UpdatedAt,
		); err != nil {
			return nil, false, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := false
	if len(tags) > limit {
		hasMore = true
		tags = tags[:limit]
	}
	return tags, hasMore, nil
}

func (s *Server) updateTag(id int64, name, slug, description string, sortOrder int) error {
	_, err := s.db.Exec(
		`UPDATE tags
		 SET name = $1, slug = $2, description = $3, sort_order = $4, updated_at = NOW()
		 WHERE id = $5`,
		name,
		slug,
		description,
		sortOrder,
		id,
	)
	return err
}

func (s *Server) deleteTag(id int64) error {
	_, err := s.db.Exec(`DELETE FROM tags WHERE id = $1`, id)
	return err
}

func (s *Server) createPost(userID string, tagID *int64, content string, createdAt time.Time) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO posts (user_id, tag_id, content, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		userID,
		tagID,
		content,
		createdAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Server) deletePost(postID int64) error {
	_, err := s.db.Exec(`DELETE FROM posts WHERE id = $1`, postID)
	return err
}

func (s *Server) addPostImage(postID int64, fileURL string, createdAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO post_images (post_id, file_url, created_at)
		 VALUES ($1, $2, $3)`,
		postID,
		fileURL,
		createdAt,
	)
	return err
}

func (s *Server) listPosts(userID string, limit, offset int) ([]Post, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(
		`SELECT p.id, p.user_id, u.username, p.tag_id, p.content, p.created_at,
		        COALESCE(l.like_count, 0) AS like_count,
		        COALESCE(r.reply_count, 0) AS reply_count,
		        (pl.user_id IS NOT NULL) AS liked_by_me
		   FROM posts p
		   JOIN users u ON u.id = p.user_id
		   LEFT JOIN (
		     SELECT post_id, COUNT(*) AS like_count
		       FROM post_likes
		      GROUP BY post_id
		   ) l ON l.post_id = p.id
		   LEFT JOIN (
		     SELECT post_id, COUNT(*) AS reply_count
		       FROM post_replies
		      GROUP BY post_id
		   ) r ON r.post_id = p.id
		   LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $1
		  ORDER BY p.created_at DESC
		  LIMIT $2 OFFSET $3`,
		userID,
		limit+1,
		offset,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	posts := make([]Post, 0, limit+1)
	postIDs := make([]int64, 0, limit+1)
	for rows.Next() {
		var post Post
		if err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Username,
			&post.TagID,
			&post.Content,
			&post.CreatedAt,
			&post.LikeCount,
			&post.ReplyCount,
			&post.LikedByMe,
		); err != nil {
			return nil, false, err
		}
		posts = append(posts, post)
		postIDs = append(postIDs, post.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := false
	if len(posts) > limit {
		hasMore = true
		posts = posts[:limit]
		postIDs = postIDs[:limit]
	}

	if len(postIDs) == 0 {
		return posts, hasMore, nil
	}

	imageRows, err := s.db.Query(
		`SELECT post_id, file_url FROM post_images
		  WHERE post_id = ANY($1)
		  ORDER BY id ASC`,
		pq.Array(postIDs),
	)
	if err != nil {
		return posts, hasMore, err
	}
	defer imageRows.Close()

	imageMap := make(map[int64][]string, len(postIDs))
	for imageRows.Next() {
		var postID int64
		var fileURL string
		if err := imageRows.Scan(&postID, &fileURL); err != nil {
			return posts, hasMore, err
		}
		imageMap[postID] = append(imageMap[postID], fileURL)
	}
	if err := imageRows.Err(); err != nil {
		return posts, hasMore, err
	}

	for i := range posts {
		posts[i].Images = imageMap[posts[i].ID]
	}

	return posts, hasMore, nil
}

func (s *Server) getPostByID(userID string, postID int64) (*Post, error) {
	var post Post
	err := s.db.QueryRow(
		`SELECT p.id, p.user_id, u.username, p.tag_id, p.content, p.created_at,
		        COALESCE(l.like_count, 0) AS like_count,
		        COALESCE(r.reply_count, 0) AS reply_count,
		        (pl.user_id IS NOT NULL) AS liked_by_me
		   FROM posts p
		   JOIN users u ON u.id = p.user_id
		   LEFT JOIN (
		     SELECT post_id, COUNT(*) AS like_count
		       FROM post_likes
		      GROUP BY post_id
		   ) l ON l.post_id = p.id
		   LEFT JOIN (
		     SELECT post_id, COUNT(*) AS reply_count
		       FROM post_replies
		      GROUP BY post_id
		   ) r ON r.post_id = p.id
		   LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $1
		  WHERE p.id = $2`,
		userID,
		postID,
	).Scan(
		&post.ID,
		&post.UserID,
		&post.Username,
		&post.TagID,
		&post.Content,
		&post.CreatedAt,
		&post.LikeCount,
		&post.ReplyCount,
		&post.LikedByMe,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	rows, err := s.db.Query(
		`SELECT file_url FROM post_images WHERE post_id = $1 ORDER BY id ASC`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		images = append(images, url)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	post.Images = images

	return &post, nil
}

func (s *Server) likePost(postID int64, userID string, createdAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO post_likes (post_id, user_id, created_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (post_id, user_id) DO NOTHING`,
		postID,
		userID,
		createdAt,
	)
	return err
}

func (s *Server) unlikePost(postID int64, userID string) error {
	_, err := s.db.Exec(`DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2`, postID, userID)
	return err
}

func (s *Server) createReply(postID int64, userID, content string, createdAt time.Time) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO post_replies (post_id, user_id, content, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		postID,
		userID,
		content,
		createdAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Server) listReplies(postID int64, limit, offset int) ([]PostReply, bool, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(
		`SELECT r.id, r.post_id, r.user_id, u.username, r.content, r.created_at
		   FROM post_replies r
		   JOIN users u ON u.id = r.user_id
		  WHERE r.post_id = $1
		  ORDER BY r.created_at ASC
		  LIMIT $2 OFFSET $3`,
		postID,
		limit+1,
		offset,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	replies := make([]PostReply, 0, limit+1)
	for rows.Next() {
		var reply PostReply
		if err := rows.Scan(
			&reply.ID,
			&reply.PostID,
			&reply.UserID,
			&reply.Username,
			&reply.Content,
			&reply.CreatedAt,
		); err != nil {
			return nil, false, err
		}
		replies = append(replies, reply)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := false
	if len(replies) > limit {
		hasMore = true
		replies = replies[:limit]
	}

	return replies, hasMore, nil
}

func normalizeChatPair(userA, userB string) (string, string) {
	if userA <= userB {
		return userA, userB
	}
	return userB, userA
}

func (s *Server) ensureChatThread(userA, userB string, createdAt time.Time) (*ChatThread, error) {
	userLow, userHigh := normalizeChatPair(userA, userB)
	var thread ChatThread
	var lastMessageAt sql.NullTime
	err := s.db.QueryRow(
		`INSERT INTO chat_threads (user_low, user_high, created_at, last_message)
		 VALUES ($1, $2, $3, '')
		 ON CONFLICT (user_low, user_high)
		 DO UPDATE SET user_low = EXCLUDED.user_low
		 RETURNING id, user_low, user_high, created_at, last_message, last_message_at`,
		userLow,
		userHigh,
		createdAt,
	).Scan(
		&thread.ID,
		&thread.UserLow,
		&thread.UserHigh,
		&thread.CreatedAt,
		&thread.LastMessage,
		&lastMessageAt,
	)
	if err != nil {
		return nil, err
	}
	if lastMessageAt.Valid {
		thread.LastMessageAt = &lastMessageAt.Time
	}
	return &thread, nil
}

func (s *Server) isChatParticipant(threadID int64, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		`SELECT EXISTS(
		   SELECT 1 FROM chat_threads
		    WHERE id = $1 AND (user_low = $2 OR user_high = $2)
		 )`,
		threadID,
		userID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (s *Server) listChatThreads(userID string, limit, offset int) ([]ChatSummary, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(
		`SELECT t.id,
		        CASE WHEN t.user_low = $1 THEN t.user_high ELSE t.user_low END AS other_id,
		        u.username,
		        t.last_message,
		        t.last_message_at,
		        t.created_at,
		        COALESCE(uc.unread_count, 0) AS unread_count
		   FROM chat_threads t
		   JOIN users u ON u.id = CASE WHEN t.user_low = $1 THEN t.user_high ELSE t.user_low END
		   LEFT JOIN chat_reads r ON r.thread_id = t.id AND r.user_id = $1
		   LEFT JOIN LATERAL (
		     SELECT COUNT(*) AS unread_count
		       FROM chat_messages m
		      WHERE m.thread_id = t.id
		        AND m.sender_id <> $1
		        AND m.deleted_at IS NULL
		        AND m.created_at > COALESCE(r.last_read_at, TIMESTAMPTZ '1970-01-01')
		   ) uc ON true
		  WHERE t.user_low = $1 OR t.user_high = $1
		  ORDER BY COALESCE(t.last_message_at, t.created_at) DESC
		  LIMIT $2 OFFSET $3`,
		userID,
		limit+1,
		offset,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	threads := make([]ChatSummary, 0, limit+1)
	for rows.Next() {
		var summary ChatSummary
		var lastMessageAt sql.NullTime
		if err := rows.Scan(
			&summary.ID,
			&summary.OtherUserID,
			&summary.OtherUsername,
			&summary.LastMessage,
			&lastMessageAt,
			&summary.CreatedAt,
			&summary.UnreadCount,
		); err != nil {
			return nil, false, err
		}
		if lastMessageAt.Valid {
			summary.LastMessageAt = &lastMessageAt.Time
		}
		threads = append(threads, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := false
	if len(threads) > limit {
		hasMore = true
		threads = threads[:limit]
	}
	return threads, hasMore, nil
}

func (s *Server) getChatSummary(userID string, threadID int64) (*ChatSummary, error) {
	var summary ChatSummary
	var lastMessageAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT t.id,
		        CASE WHEN t.user_low = $1 THEN t.user_high ELSE t.user_low END AS other_id,
		        u.username,
		        t.last_message,
		        t.last_message_at,
		        t.created_at,
		        COALESCE(uc.unread_count, 0) AS unread_count
		   FROM chat_threads t
		   JOIN users u ON u.id = CASE WHEN t.user_low = $1 THEN t.user_high ELSE t.user_low END
		   LEFT JOIN chat_reads r ON r.thread_id = t.id AND r.user_id = $1
		   LEFT JOIN LATERAL (
		     SELECT COUNT(*) AS unread_count
		       FROM chat_messages m
		      WHERE m.thread_id = t.id
		        AND m.sender_id <> $1
		        AND m.deleted_at IS NULL
		        AND m.created_at > COALESCE(r.last_read_at, TIMESTAMPTZ '1970-01-01')
		   ) uc ON true
		  WHERE t.id = $2 AND (t.user_low = $1 OR t.user_high = $1)`,
		userID,
		threadID,
	).Scan(
		&summary.ID,
		&summary.OtherUserID,
		&summary.OtherUsername,
		&summary.LastMessage,
		&lastMessageAt,
		&summary.CreatedAt,
		&summary.UnreadCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if lastMessageAt.Valid {
		summary.LastMessageAt = &lastMessageAt.Time
	}
	return &summary, nil
}

func (s *Server) listChatMessages(threadID int64, limit, offset int) ([]ChatMessage, bool, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(
		`SELECT m.id, m.thread_id, m.sender_id, u.username, m.content, m.created_at, m.deleted_at, m.deleted_by
		   FROM chat_messages m
		   JOIN users u ON u.id = m.sender_id
		  WHERE m.thread_id = $1
		  ORDER BY m.created_at ASC
		  LIMIT $2 OFFSET $3`,
		threadID,
		limit+1,
		offset,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	messages := make([]ChatMessage, 0, limit+1)
	for rows.Next() {
		var msg ChatMessage
		var deletedAt sql.NullTime
		var deletedBy sql.NullString
		if err := rows.Scan(
			&msg.ID,
			&msg.ThreadID,
			&msg.SenderID,
			&msg.SenderUsername,
			&msg.Content,
			&msg.CreatedAt,
			&deletedAt,
			&deletedBy,
		); err != nil {
			return nil, false, err
		}
		if deletedAt.Valid {
			msg.DeletedAt = &deletedAt.Time
			msg.Deleted = true
			msg.Content = ""
		}
		if deletedBy.Valid {
			msg.DeletedBy = deletedBy.String
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := false
	if len(messages) > limit {
		hasMore = true
		messages = messages[:limit]
	}
	return messages, hasMore, nil
}

func (s *Server) markChatRead(threadID int64, userID string, readAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO chat_reads (thread_id, user_id, last_read_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (thread_id, user_id)
		 DO UPDATE SET last_read_at = EXCLUDED.last_read_at`,
		threadID,
		userID,
		readAt,
	)
	return err
}

func (s *Server) createChatMessage(threadID int64, senderID, content string, createdAt time.Time) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var id int64
	err = tx.QueryRow(
		`INSERT INTO chat_messages (thread_id, sender_id, content, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		threadID,
		senderID,
		content,
		createdAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	if _, err = tx.Exec(
		`UPDATE chat_threads
		    SET last_message = $1, last_message_at = $2
		  WHERE id = $3`,
		content,
		createdAt,
		threadID,
	); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Server) deleteChatMessage(threadID, messageID int64, userID string, deletedAt time.Time) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var senderID string
	var existingDeletedAt sql.NullTime
	err = tx.QueryRow(
		`SELECT sender_id, deleted_at
		   FROM chat_messages
		  WHERE id = $1 AND thread_id = $2`,
		messageID,
		threadID,
	).Scan(&senderID, &existingDeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if senderID != userID {
		return false, errNotMessageOwner
	}
	if existingDeletedAt.Valid {
		return false, nil
	}

	if _, err = tx.Exec(
		`UPDATE chat_messages
		    SET deleted_at = $1, deleted_by = $2
		  WHERE id = $3`,
		deletedAt,
		userID,
		messageID,
	); err != nil {
		return false, err
	}

	var latestID int64
	err = tx.QueryRow(
		`SELECT id FROM chat_messages
		  WHERE thread_id = $1
		  ORDER BY created_at DESC, id DESC
		  LIMIT 1`,
		threadID,
	).Scan(&latestID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if latestID == messageID {
		if _, err = tx.Exec(
			`UPDATE chat_threads
			    SET last_message = $1, last_message_at = $2
			  WHERE id = $3`,
			"消息已撤回",
			deletedAt,
			threadID,
		); err != nil {
			return false, err
		}
	}

	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
