package dock

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var (
	errEmailExists       = errors.New("email already exists")
	errNotMessageOwner   = errors.New("not message owner")
	errChatReplyRequired = errors.New("chat reply required")
	errTaskNotFound      = errors.New("task not found")
	errTaskClosed        = errors.New("task application closed")
	errTaskApplyEnded    = errors.New("task application deadline passed")
	errTaskSelfApply     = errors.New("task owner cannot apply")
	errTaskForbidden     = errors.New("task forbidden")
)

type User struct {
	ID         string     `json:"id"`
	Username   string     `json:"username"`
	Email      string     `json:"email"`
	Password   string     `json:"-"` // password_hash
	Role       string     `json:"role"`
	Bio        string     `json:"bio"`
	IconURL    string     `json:"icon_url"`
	IsOnline   bool       `json:"is_online"`
	DeviceType string     `json:"device_type"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Session struct {
	ID         string
	UserID     string
	Username   string
	Role       string
	DeviceType string
	DeviceID   string
	PushToken  string
	ExpiresAt  time.Time
}

type UserDevice struct {
	ID           int64
	UserID       string
	DeviceType   string
	DeviceID     string
	PushToken    string
	PushEnabled  bool
	AppVersion   string
	IsOnline     bool
	LastLoginAt  time.Time
	LastSeenAt   *time.Time
	LastActiveAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PushDelivery struct {
	ID           int64
	MessageID    int64
	UserID       string
	DeviceID     string
	PushToken    string
	Provider     string
	Status       string
	APNSID       string
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type MarkdownEntry struct {
	ID         int64     `json:"id"`
	UserID     string    `json:"user_id"`
	Title      string    `json:"title"`
	FilePath   string    `json:"file_path"`
	IsPublic   bool      `json:"is_public"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type PublicMarkdownEntry struct {
	ID         int64     `json:"id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	UserIcon   string    `json:"user_icon"`
	Title      string    `json:"title"`
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
	DeviceType  string    `json:"device_type"`
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

type SiteSettings struct {
	Name              string                `json:"name"`
	Description       string                `json:"description"`
	IconURL           string                `json:"icon_url"`
	ApplePushDevCert  *ApplePushCertificate `json:"apple_push_dev_cert,omitempty"`
	ApplePushProdCert *ApplePushCertificate `json:"apple_push_prod_cert,omitempty"`
	SystemInfo        *SystemInfo           `json:"system_info,omitempty"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

type SystemInfo struct {
	GitTagVersion     string `json:"git_tag_version"`
	OS                string `json:"os"`
	CPUArch           string `json:"cpu_arch"`
	PartitionPath     string `json:"partition_path"`
	PartitionCapacity string `json:"partition_capacity"`
}

type LLMConfig struct {
	ID           int64     `json:"id"`
	OwnerUserID  string    `json:"owner_user_id"`
	ShareID      string    `json:"share_id"`
	Shared       bool      `json:"shared"`
	Name         string    `json:"name"`
	BaseURL      string    `json:"base_url"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt"`
	HasAPIKey    bool      `json:"has_api_key"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BotUser struct {
	ID           int64     `json:"id"`
	OwnerUserID  string    `json:"owner_user_id"`
	BotUserID    string    `json:"bot_user_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SystemPrompt string    `json:"system_prompt"`
	LLMConfigID  int64     `json:"llm_config_id"`
	ConfigName   string    `json:"config_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ApplePushCertificate struct {
	Environment string     `json:"environment"`
	FileName    string     `json:"file_name"`
	FileURL     string     `json:"file_url"`
	UploadedAt  *time.Time `json:"uploaded_at,omitempty"`
}

type Post struct {
	ID         int64       `json:"id"`
	UserID     string      `json:"user_id"`
	Username   string      `json:"username"`
	UserIcon   string      `json:"user_icon"`
	TagID      *int64      `json:"tag_id,omitempty"`
	PostType   string      `json:"post_type"`
	Content    string      `json:"content"`
	CreatedAt  time.Time   `json:"created_at"`
	LikeCount  int         `json:"like_count"`
	ReplyCount int         `json:"reply_count"`
	LikedByMe  bool        `json:"liked_by_me"`
	Images     []string    `json:"images"`
	ImageItems []PostImage `json:"image_items,omitempty"`
	Videos     []string    `json:"videos"`
	VideoItems []PostVideo `json:"video_items,omitempty"`
	Task       *TaskPost   `json:"task,omitempty"`
}

type PostImage struct {
	OriginalURL string `json:"original_url"`
	MediumURL   string `json:"medium_url,omitempty"`
	SmallURL    string `json:"small_url,omitempty"`
}

type PostVideo struct {
	URL       string `json:"url"`
	PosterURL string `json:"poster_url,omitempty"`
}

type PostReply struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	UserIcon  string    `json:"user_icon"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatThread struct {
	ID            int64      `json:"id"`
	UserLow       string     `json:"user_low"`
	UserHigh      string     `json:"user_high"`
	CreatedAt     time.Time  `json:"created_at"`
	LastMessage   string     `json:"last_message"`
	LastMessageID *int64     `json:"last_message_id,omitempty"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
}

type ChatSummary struct {
	ID                   int64      `json:"id"`
	OtherUserID          string     `json:"other_user_id"`
	OtherUsername        string     `json:"other_username"`
	OtherUserIcon        string     `json:"other_user_icon"`
	OtherUserOnline      bool       `json:"other_user_online"`
	OtherUserDeviceType  string     `json:"other_user_device_type"`
	OtherUserLastSeenAt  *time.Time `json:"other_user_last_seen_at,omitempty"`
	LastMessage          string     `json:"last_message"`
	LastMessageAt        *time.Time `json:"last_message_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UnreadCount          int        `json:"unread_count"`
	IsImplicitFriend     bool       `json:"is_implicit_friend"`
	ReplyRequired        bool       `json:"reply_required"`
	ReplyRequiredMessage string     `json:"reply_required_message"`
}

type ChatMessageAttachment struct {
	URL          string `json:"url"`
	FileName     string `json:"file_name"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mime_type"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
}

type ChatMessage struct {
	ID              int64                  `json:"id"`
	ThreadID        int64                  `json:"thread_id"`
	LLMThreadID     *int64                 `json:"llm_thread_id,omitempty"`
	SenderID        string                 `json:"sender_id"`
	SenderUsername  string                 `json:"sender_username"`
	SenderIcon      string                 `json:"sender_icon"`
	MessageType     string                 `json:"message_type"`
	Failed          bool                   `json:"failed"`
	Content         string                 `json:"content"`
	MarkdownEntryID *int64                 `json:"markdown_entry_id,omitempty"`
	MarkdownTitle   string                 `json:"markdown_title,omitempty"`
	Attachment      *ChatMessageAttachment `json:"attachment,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	DeletedAt       *time.Time             `json:"deleted_at,omitempty"`
	DeletedBy       string                 `json:"deleted_by,omitempty"`
	Deleted         bool                   `json:"deleted"`
}

type LLMThread struct {
	ID            int64      `json:"id"`
	ChatThreadID  int64      `json:"chat_thread_id"`
	OwnerUserID   string     `json:"owner_user_id"`
	BotUserID     string     `json:"bot_user_id"`
	LLMConfigID   *int64     `json:"llm_config_id,omitempty"`
	ConfigName    string     `json:"config_name,omitempty"`
	ConfigModel   string     `json:"config_model,omitempty"`
	Title         string     `json:"title"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
}

type WebAuthnCredentialSummary struct {
	CredentialID string    `json:"credential_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TaskPost struct {
	PostID                int64      `json:"post_id"`
	Location              string     `json:"location,omitempty"`
	StartAt               time.Time  `json:"start_at"`
	EndAt                 time.Time  `json:"end_at"`
	WorkingHours          string     `json:"working_hours"`
	ApplyDeadline         time.Time  `json:"apply_deadline"`
	ApplicationStatus     string     `json:"application_status"`
	SelectedApplicantID   string     `json:"selected_applicant_id,omitempty"`
	SelectedApplicantName string     `json:"selected_applicant_name,omitempty"`
	SelectedAt            *time.Time `json:"selected_at,omitempty"`
	InvitationTemplate    string     `json:"invitation_template,omitempty"`
	InvitationSentAt      *time.Time `json:"invitation_sent_at,omitempty"`
	ApplicantCount        int        `json:"applicant_count"`
	AppliedByMe           bool       `json:"applied_by_me"`
	CanApply              bool       `json:"can_apply"`
	CanManage             bool       `json:"can_manage"`
	SelectedByMe          bool       `json:"selected_by_me"`
	CanViewResults        bool       `json:"can_view_results"`
	CanSubmitResult       bool       `json:"can_submit_result"`
}

type TaskApplication struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	UserIcon  string    `json:"user_icon"`
	AppliedAt time.Time `json:"applied_at"`
}

type TaskResult struct {
	ID         int64       `json:"id"`
	PostID     int64       `json:"post_id"`
	UserID     string      `json:"user_id"`
	Username   string      `json:"username"`
	UserIcon   string      `json:"user_icon"`
	Note       string      `json:"note"`
	CreatedAt  time.Time   `json:"created_at"`
	Images     []string    `json:"images"`
	Videos     []string    `json:"videos"`
	VideoItems []PostVideo `json:"video_items,omitempty"`
}

type ProfileRecommendation struct {
	ID             int64     `json:"id"`
	TargetUserID   string    `json:"target_user_id"`
	AuthorUserID   string    `json:"author_user_id"`
	AuthorUsername string    `json:"author_username"`
	AuthorUserIcon string    `json:"author_user_icon"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UserProfileDetail struct {
	UserID          string                  `json:"user_id"`
	Username        string                  `json:"username"`
	Email           string                  `json:"email,omitempty"`
	IconURL         string                  `json:"icon_url"`
	Bio             string                  `json:"bio"`
	CreatedAt       time.Time               `json:"created_at"`
	IsMe            bool                    `json:"is_me"`
	CanRecommend    bool                    `json:"can_recommend"`
	IBlockedUser    bool                    `json:"i_blocked_user"`
	BlockedMe       bool                    `json:"blocked_me"`
	Recommendations []ProfileRecommendation `json:"recommendations"`
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
	bio TEXT NOT NULL DEFAULT '',
	icon_url TEXT NOT NULL DEFAULT '',
	is_online BOOLEAN NOT NULL DEFAULT FALSE,
	last_active_device_type TEXT NOT NULL DEFAULT 'browser',
	last_seen_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE users
	ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS bio TEXT NOT NULL DEFAULT '';
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS icon_url TEXT NOT NULL DEFAULT '';
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS is_online BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS last_active_device_type TEXT NOT NULL DEFAULT 'browser';
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS profile_recommendations (
	id BIGSERIAL PRIMARY KEY,
	target_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	author_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	content TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS user_blocks (
	blocker_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	blocked_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (blocker_user_id, blocked_user_id)
);

CREATE TABLE IF NOT EXISTS markdown_entries (
	id BIGSERIAL PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	file_path TEXT NOT NULL,
	is_public BOOLEAN NOT NULL DEFAULT FALSE,
	uploaded_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE markdown_entries
	ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT FALSE;

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
	device_type TEXT NOT NULL DEFAULT 'browser',
	logged_in_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE login_records
	ADD COLUMN IF NOT EXISTS device_type TEXT NOT NULL DEFAULT 'browser';

CREATE TABLE IF NOT EXISTS user_devices (
	id BIGSERIAL PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	device_type TEXT NOT NULL DEFAULT 'browser',
	device_id TEXT NOT NULL DEFAULT '',
	push_token TEXT NOT NULL DEFAULT '',
	push_enabled BOOLEAN NOT NULL DEFAULT TRUE,
	app_version TEXT NOT NULL DEFAULT '',
	is_online BOOLEAN NOT NULL DEFAULT FALSE,
	last_login_at TIMESTAMPTZ NOT NULL,
	last_seen_at TIMESTAMPTZ,
	last_active_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	UNIQUE (user_id, device_type, device_id)
);

ALTER TABLE user_devices
	ADD COLUMN IF NOT EXISTS device_id TEXT NOT NULL DEFAULT '';
ALTER TABLE user_devices
	ADD COLUMN IF NOT EXISTS push_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE user_devices
	ADD COLUMN IF NOT EXISTS app_version TEXT NOT NULL DEFAULT '';
ALTER TABLE user_devices
	ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ;
UPDATE user_devices
   SET device_id = CASE
       WHEN COALESCE(device_id, '') <> '' THEN device_id
       WHEN device_type = 'ios' THEN 'default:ios'
       WHEN device_type = 'android' THEN 'default:android'
       ELSE 'default:browser'
   END
 WHERE COALESCE(device_id, '') = '';
ALTER TABLE user_devices
	DROP CONSTRAINT IF EXISTS user_devices_user_id_device_type_key;

CREATE TABLE IF NOT EXISTS push_deliveries (
	id BIGSERIAL PRIMARY KEY,
	message_id BIGINT NOT NULL,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	device_id TEXT NOT NULL DEFAULT '',
	push_token TEXT NOT NULL DEFAULT '',
	provider TEXT NOT NULL DEFAULT 'apns',
	status TEXT NOT NULL,
	apns_id TEXT NOT NULL DEFAULT '',
	error_message TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS chat_member_state (
	thread_id BIGINT NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	mute_until TIMESTAMPTZ,
	is_muted BOOLEAN NOT NULL DEFAULT FALSE,
	last_opened_at TIMESTAMPTZ,
	last_delivered_message_id BIGINT,
	last_push_message_id BIGINT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (thread_id, user_id)
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

CREATE TABLE IF NOT EXISTS site_settings (
	id INT PRIMARY KEY,
	name TEXT NOT NULL DEFAULT 'Polar-',
	description TEXT NOT NULL DEFAULT '',
	icon_url TEXT NOT NULL DEFAULT '',
	apple_push_dev_cert_url TEXT NOT NULL DEFAULT '',
	apple_push_dev_cert_name TEXT NOT NULL DEFAULT '',
	apple_push_dev_cert_uploaded_at TIMESTAMPTZ,
	apple_push_prod_cert_url TEXT NOT NULL DEFAULT '',
	apple_push_prod_cert_name TEXT NOT NULL DEFAULT '',
	apple_push_prod_cert_uploaded_at TIMESTAMPTZ,
	updated_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE site_settings
	ADD COLUMN IF NOT EXISTS apple_push_dev_cert_url TEXT NOT NULL DEFAULT '';
ALTER TABLE site_settings
	ADD COLUMN IF NOT EXISTS apple_push_dev_cert_name TEXT NOT NULL DEFAULT '';
ALTER TABLE site_settings
	ADD COLUMN IF NOT EXISTS apple_push_dev_cert_uploaded_at TIMESTAMPTZ;
ALTER TABLE site_settings
	ADD COLUMN IF NOT EXISTS apple_push_prod_cert_url TEXT NOT NULL DEFAULT '';
ALTER TABLE site_settings
	ADD COLUMN IF NOT EXISTS apple_push_prod_cert_name TEXT NOT NULL DEFAULT '';
ALTER TABLE site_settings
	ADD COLUMN IF NOT EXISTS apple_push_prod_cert_uploaded_at TIMESTAMPTZ;

INSERT INTO site_settings (
	id, name, description, icon_url,
	apple_push_dev_cert_url, apple_push_dev_cert_name, apple_push_dev_cert_uploaded_at,
	apple_push_prod_cert_url, apple_push_prod_cert_name, apple_push_prod_cert_uploaded_at,
	updated_at
)
VALUES (1, 'Polar-', '', '', '', '', NULL, '', '', NULL, NOW())
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS llm_configs (
	id BIGSERIAL PRIMARY KEY,
	owner_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	base_url TEXT NOT NULL,
	model TEXT NOT NULL,
	api_key TEXT NOT NULL DEFAULT '',
	system_prompt TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS bot_users (
	id BIGSERIAL PRIMARY KEY,
	owner_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	bot_user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	system_prompt TEXT NOT NULL DEFAULT '',
	llm_config_id BIGINT NOT NULL REFERENCES llm_configs(id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE bot_users
	ADD COLUMN IF NOT EXISTS system_prompt TEXT NOT NULL DEFAULT '';

ALTER TABLE llm_configs
	ADD COLUMN IF NOT EXISTS share_id TEXT NOT NULL DEFAULT '';
ALTER TABLE llm_configs
	ADD COLUMN IF NOT EXISTS shared BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS posts (
	id BIGSERIAL PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	tag_id BIGINT REFERENCES tags(id) ON DELETE SET NULL,
	post_type TEXT NOT NULL DEFAULT 'standard',
	content TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE posts
	ADD COLUMN IF NOT EXISTS post_type TEXT NOT NULL DEFAULT 'standard';

CREATE TABLE IF NOT EXISTS task_posts (
	post_id BIGINT PRIMARY KEY REFERENCES posts(id) ON DELETE CASCADE,
	location TEXT NOT NULL DEFAULT '',
	start_at TIMESTAMPTZ NOT NULL,
	end_at TIMESTAMPTZ NOT NULL,
	working_hours TEXT NOT NULL,
	apply_deadline TIMESTAMPTZ NOT NULL,
	application_status TEXT NOT NULL DEFAULT 'open',
	selected_applicant_id TEXT REFERENCES users(id) ON DELETE SET NULL,
	selected_at TIMESTAMPTZ,
	invitation_template TEXT NOT NULL DEFAULT '',
	invitation_sent_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS task_applications (
	id BIGSERIAL PRIMARY KEY,
	post_id BIGINT NOT NULL REFERENCES task_posts(post_id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	applied_at TIMESTAMPTZ NOT NULL,
	withdrawn_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS task_results (
	id BIGSERIAL PRIMARY KEY,
	post_id BIGINT NOT NULL REFERENCES task_posts(post_id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	note TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS task_result_images (
	id BIGSERIAL PRIMARY KEY,
	result_id BIGINT NOT NULL REFERENCES task_results(id) ON DELETE CASCADE,
	file_url TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS task_result_videos (
	id BIGSERIAL PRIMARY KEY,
	result_id BIGINT NOT NULL REFERENCES task_results(id) ON DELETE CASCADE,
	file_url TEXT NOT NULL,
	poster_url TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS post_images (
	id BIGSERIAL PRIMARY KEY,
	post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
	file_url TEXT NOT NULL,
	small_url TEXT NOT NULL DEFAULT '',
	medium_url TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE post_images
	ADD COLUMN IF NOT EXISTS small_url TEXT NOT NULL DEFAULT '';

ALTER TABLE post_images
	ADD COLUMN IF NOT EXISTS medium_url TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS post_videos (
	id BIGSERIAL PRIMARY KEY,
	post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
	file_url TEXT NOT NULL,
	poster_url TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE post_videos
	ADD COLUMN IF NOT EXISTS poster_url TEXT NOT NULL DEFAULT '';

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
	last_message_id BIGINT,
	last_message_at TIMESTAMPTZ
);

ALTER TABLE chat_threads
	ADD COLUMN IF NOT EXISTS last_message_id BIGINT;

CREATE TABLE IF NOT EXISTS llm_threads (
	id BIGSERIAL PRIMARY KEY,
	chat_thread_id BIGINT NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
	owner_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	bot_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	llm_config_id BIGINT REFERENCES llm_configs(id) ON DELETE SET NULL,
	title TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	last_message_at TIMESTAMPTZ
);

ALTER TABLE llm_threads
	ADD COLUMN IF NOT EXISTS llm_config_id BIGINT REFERENCES llm_configs(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS chat_messages (
	id BIGSERIAL PRIMARY KEY,
	thread_id BIGINT NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
	llm_thread_id BIGINT REFERENCES llm_threads(id) ON DELETE SET NULL,
	sender_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	message_type TEXT NOT NULL DEFAULT 'text',
	failed BOOLEAN NOT NULL DEFAULT FALSE,
	content TEXT NOT NULL,
	markdown_entry_id BIGINT REFERENCES markdown_entries(id) ON DELETE SET NULL,
	markdown_title TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE chat_messages
	ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE chat_messages
	ADD COLUMN IF NOT EXISTS deleted_by TEXT;

ALTER TABLE chat_messages
	ADD COLUMN IF NOT EXISTS message_type TEXT NOT NULL DEFAULT 'text';

ALTER TABLE chat_messages
	ADD COLUMN IF NOT EXISTS failed BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE chat_messages
	ADD COLUMN IF NOT EXISTS markdown_entry_id BIGINT REFERENCES markdown_entries(id) ON DELETE SET NULL;

ALTER TABLE chat_messages
	ADD COLUMN IF NOT EXISTS markdown_title TEXT NOT NULL DEFAULT '';

ALTER TABLE chat_messages
	ADD COLUMN IF NOT EXISTS llm_thread_id BIGINT REFERENCES llm_threads(id) ON DELETE SET NULL;

ALTER TABLE chat_messages
	ADD COLUMN IF NOT EXISTS attachment TEXT;

CREATE TABLE IF NOT EXISTS chat_reads (
	thread_id BIGINT NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	last_read_at TIMESTAMPTZ NOT NULL,
	last_read_message_id BIGINT,
	PRIMARY KEY (thread_id, user_id)
);

ALTER TABLE chat_reads
	ADD COLUMN IF NOT EXISTS last_read_message_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_markdown_entries_user_id ON markdown_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON webauthn_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_login_records_user_id_logged_in_at ON login_records(user_id, logged_in_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_devices_user_id ON user_devices(user_id);
CREATE INDEX IF NOT EXISTS idx_user_devices_push_token ON user_devices(push_token);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_devices_user_device ON user_devices(user_id, device_type, device_id);
CREATE INDEX IF NOT EXISTS idx_push_deliveries_message_id ON push_deliveries(message_id);
CREATE INDEX IF NOT EXISTS idx_push_deliveries_user_id_created_at ON push_deliveries(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_llm_configs_owner_user_id ON llm_configs(owner_user_id, updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_configs_share_id ON llm_configs(share_id) WHERE share_id <> '';
CREATE INDEX IF NOT EXISTS idx_bot_users_owner_user_id ON bot_users(owner_user_id, updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_profile_recommendations_target_author ON profile_recommendations(target_user_id, author_user_id);
CREATE INDEX IF NOT EXISTS idx_profile_recommendations_target_updated_at ON profile_recommendations(target_user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_blocks_blocked_user_id ON user_blocks(blocked_user_id, blocker_user_id);
CREATE INDEX IF NOT EXISTS idx_tags_sort_order_created_at ON tags(sort_order DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_post_images_post_id ON post_images(post_id);
CREATE INDEX IF NOT EXISTS idx_post_videos_post_id ON post_videos(post_id);
CREATE INDEX IF NOT EXISTS idx_post_likes_post_id ON post_likes(post_id);
CREATE INDEX IF NOT EXISTS idx_post_replies_post_id ON post_replies(post_id);
CREATE INDEX IF NOT EXISTS idx_task_posts_apply_deadline ON task_posts(apply_deadline DESC);
CREATE INDEX IF NOT EXISTS idx_task_applications_post_id ON task_applications(post_id, applied_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_task_applications_active_pair
	ON task_applications(post_id, user_id)
	WHERE withdrawn_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_task_results_post_id_created_at ON task_results(post_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_task_result_images_result_id ON task_result_images(result_id);
CREATE INDEX IF NOT EXISTS idx_task_result_videos_result_id ON task_result_videos(result_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_threads_pair ON chat_threads(user_low, user_high);
CREATE INDEX IF NOT EXISTS idx_chat_threads_last_message_at ON chat_threads(last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_llm_threads_chat_thread_id_updated_at ON llm_threads(chat_thread_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_thread_id_created_at ON chat_messages(thread_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_llm_thread_id_created_at ON chat_messages(llm_thread_id, created_at DESC);
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
		`INSERT INTO login_records (user_id, ip_address, country, region, city, login_method, device_type, logged_in_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		record.UserID,
		record.IPAddress,
		record.Country,
		record.Region,
		record.City,
		record.LoginMethod,
		record.DeviceType,
		record.LoggedInAt,
	)
	return err
}

func (s *Server) listLoginRecords(userID string, limit int) ([]LoginRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(
		`SELECT id, user_id, ip_address, country, region, city, login_method, device_type, logged_in_at
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
			&record.DeviceType,
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
	var lastSeenAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, username, email, password_hash, role, bio, icon_url, is_online, last_active_device_type, last_seen_at, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.Bio, &user.IconURL, &user.IsOnline, &user.DeviceType, &lastSeenAt, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if lastSeenAt.Valid {
		user.LastSeenAt = &lastSeenAt.Time
	}
	return &user, nil
}

func (s *Server) getUserByID(userID string) (*User, error) {
	var user User
	var lastSeenAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, username, email, password_hash, role, bio, icon_url, is_online, last_active_device_type, last_seen_at, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.Bio, &user.IconURL, &user.IsOnline, &user.DeviceType, &lastSeenAt, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if lastSeenAt.Valid {
		user.LastSeenAt = &lastSeenAt.Time
	}
	return &user, nil
}

func (s *Server) getUserByUsername(username string) (*User, error) {
	var user User
	var lastSeenAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, username, email, password_hash, role, bio, icon_url, is_online, last_active_device_type, last_seen_at, created_at FROM users WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.Bio, &user.IconURL, &user.IsOnline, &user.DeviceType, &lastSeenAt, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if lastSeenAt.Valid {
		user.LastSeenAt = &lastSeenAt.Time
	}
	return &user, nil
}

func (s *Server) listLLMConfigs(ownerUserID string) ([]LLMConfig, error) {
	rows, err := s.db.Query(
		`SELECT id, owner_user_id, share_id, shared, name, base_url, model, system_prompt, (api_key <> '') AS has_api_key, created_at, updated_at
		   FROM llm_configs
		  WHERE owner_user_id = $1
		  ORDER BY updated_at DESC, id DESC`,
		ownerUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]LLMConfig, 0)
	for rows.Next() {
		var item LLMConfig
		if err := rows.Scan(&item.ID, &item.OwnerUserID, &item.ShareID, &item.Shared, &item.Name, &item.BaseURL, &item.Model, &item.SystemPrompt, &item.HasAPIKey, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) getLLMConfigForOwner(ownerUserID string, id int64) (*LLMConfig, string, error) {
	var item LLMConfig
	var apiKey string
	err := s.db.QueryRow(
		`SELECT id, owner_user_id, share_id, shared, name, base_url, model, api_key, system_prompt, (api_key <> '') AS has_api_key, created_at, updated_at
		   FROM llm_configs
		  WHERE id = $1 AND owner_user_id = $2`,
		id,
		ownerUserID,
	).Scan(&item.ID, &item.OwnerUserID, &item.ShareID, &item.Shared, &item.Name, &item.BaseURL, &item.Model, &apiKey, &item.SystemPrompt, &item.HasAPIKey, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", err
	}
	return &item, apiKey, nil
}

func (s *Server) getLLMConfigForBot(botUserID string) (*LLMConfig, string, error) {
	var item LLMConfig
	var apiKey string
	err := s.db.QueryRow(
		`SELECT c.id, c.owner_user_id, c.share_id, c.shared, c.name, c.base_url, c.model, c.api_key, c.system_prompt, (c.api_key <> '') AS has_api_key, c.created_at, c.updated_at
		   FROM bot_users b
		   JOIN llm_configs c ON c.id = b.llm_config_id
		  WHERE b.bot_user_id = $1`,
		botUserID,
	).Scan(&item.ID, &item.OwnerUserID, &item.ShareID, &item.Shared, &item.Name, &item.BaseURL, &item.Model, &apiKey, &item.SystemPrompt, &item.HasAPIKey, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", err
	}
	return &item, apiKey, nil
}

func (s *Server) createLLMConfig(ownerUserID, name, baseURL, model, apiKey, systemPrompt, shareID string, shared bool, now time.Time) (*LLMConfig, error) {
	var item LLMConfig
	err := s.db.QueryRow(
		`INSERT INTO llm_configs (owner_user_id, share_id, shared, name, base_url, model, api_key, system_prompt, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		 RETURNING id, owner_user_id, share_id, shared, name, base_url, model, system_prompt, (api_key <> '') AS has_api_key, created_at, updated_at`,
		ownerUserID, shareID, shared, name, baseURL, model, apiKey, systemPrompt, now,
	).Scan(&item.ID, &item.OwnerUserID, &item.ShareID, &item.Shared, &item.Name, &item.BaseURL, &item.Model, &item.SystemPrompt, &item.HasAPIKey, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Server) updateLLMConfig(ownerUserID string, id int64, name, baseURL, model, apiKey, systemPrompt string, shared, replaceAPIKey bool, now time.Time) (*LLMConfig, error) {
	query := `UPDATE llm_configs
	             SET name = $3,
	                 base_url = $4,
	                 model = $5,
	                 system_prompt = $6,
	                 shared = $7,
	                 updated_at = $8`
	args := []any{id, ownerUserID, name, baseURL, model, systemPrompt, shared, now}
	if replaceAPIKey {
		query += `, api_key = $9`
		args = append(args, apiKey)
	}
	query += ` WHERE id = $1 AND owner_user_id = $2
	           RETURNING id, owner_user_id, share_id, shared, name, base_url, model, system_prompt, (api_key <> '') AS has_api_key, created_at, updated_at`
	var item LLMConfig
	err := s.db.QueryRow(query, args...).Scan(&item.ID, &item.OwnerUserID, &item.ShareID, &item.Shared, &item.Name, &item.BaseURL, &item.Model, &item.SystemPrompt, &item.HasAPIKey, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *Server) deleteLLMConfig(ownerUserID string, id int64) (bool, error) {
	result, err := s.db.Exec(`DELETE FROM llm_configs WHERE id = $1 AND owner_user_id = $2`, id, ownerUserID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *Server) listAvailableLLMConfigs(ownerUserID string) ([]LLMConfig, error) {
	rows, err := s.db.Query(
		`SELECT id, owner_user_id, share_id, shared, name, base_url, model, system_prompt, (api_key <> '') AS has_api_key, created_at, updated_at
		   FROM llm_configs
		  WHERE owner_user_id = $1 OR shared = TRUE
		  ORDER BY (owner_user_id = $1) DESC, updated_at DESC, id DESC`,
		ownerUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]LLMConfig, 0)
	for rows.Next() {
		var item LLMConfig
		if err := rows.Scan(&item.ID, &item.OwnerUserID, &item.ShareID, &item.Shared, &item.Name, &item.BaseURL, &item.Model, &item.SystemPrompt, &item.HasAPIKey, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) getLLMConfigByShareID(shareID string) (*LLMConfig, error) {
	var item LLMConfig
	err := s.db.QueryRow(
		`SELECT id, owner_user_id, share_id, shared, name, base_url, model, system_prompt, (api_key <> '') AS has_api_key, created_at, updated_at
		   FROM llm_configs
		  WHERE share_id = $1 AND share_id <> ''`,
		shareID,
	).Scan(&item.ID, &item.OwnerUserID, &item.ShareID, &item.Shared, &item.Name, &item.BaseURL, &item.Model, &item.SystemPrompt, &item.HasAPIKey, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *Server) listBotUsers(ownerUserID string) ([]BotUser, error) {
	rows, err := s.db.Query(
		`SELECT b.id, b.owner_user_id, b.bot_user_id, b.name, b.description, b.system_prompt, b.llm_config_id, c.name, b.created_at, b.updated_at
		   FROM bot_users b
		   JOIN llm_configs c ON c.id = b.llm_config_id
		  WHERE b.owner_user_id = $1
		  ORDER BY b.updated_at DESC, b.id DESC`,
		ownerUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BotUser, 0)
	for rows.Next() {
		var item BotUser
		if err := rows.Scan(&item.ID, &item.OwnerUserID, &item.BotUserID, &item.Name, &item.Description, &item.SystemPrompt, &item.LLMConfigID, &item.ConfigName, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) getBotUserForOwner(ownerUserID string, id int64) (*BotUser, error) {
	var item BotUser
	err := s.db.QueryRow(
		`SELECT b.id, b.owner_user_id, b.bot_user_id, b.name, b.description, b.system_prompt, b.llm_config_id, c.name, b.created_at, b.updated_at
		   FROM bot_users b
		   JOIN llm_configs c ON c.id = b.llm_config_id
		  WHERE b.id = $1 AND b.owner_user_id = $2`,
		id, ownerUserID,
	).Scan(&item.ID, &item.OwnerUserID, &item.BotUserID, &item.Name, &item.Description, &item.SystemPrompt, &item.LLMConfigID, &item.ConfigName, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *Server) createBotUser(ownerUserID, name, description, systemPrompt string, llmConfigID int64, now time.Time) (*BotUser, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var configName string
	err = tx.QueryRow(`SELECT name FROM llm_configs WHERE id = $1 AND (owner_user_id = $2 OR shared = TRUE)`, llmConfigID, ownerUserID).Scan(&configName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	botUserID := "bot_" + generateSessionID()[:16]
	botEmail := botUserID + "@local.polar"
	password, err := hashPassword(generateSessionID())
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(
		`INSERT INTO users (id, username, email, password_hash, role, bio, icon_url, is_online, last_active_device_type, last_seen_at, created_at)
		 VALUES ($1, $2, $3, $4, 'bot', $5, '', FALSE, 'browser', NULL, $6)`,
		botUserID, name, botEmail, password, description, now,
	); err != nil {
		return nil, err
	}

	var item BotUser
	err = tx.QueryRow(
		`INSERT INTO bot_users (owner_user_id, bot_user_id, name, description, system_prompt, llm_config_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		 RETURNING id, owner_user_id, bot_user_id, name, description, system_prompt, llm_config_id, created_at, updated_at`,
		ownerUserID, botUserID, name, description, systemPrompt, llmConfigID, now,
	).Scan(&item.ID, &item.OwnerUserID, &item.BotUserID, &item.Name, &item.Description, &item.SystemPrompt, &item.LLMConfigID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	item.ConfigName = configName
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Server) updateBotUser(ownerUserID string, id int64, name, description, systemPrompt string, llmConfigID int64, now time.Time) (*BotUser, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var botUserID string
	err = tx.QueryRow(`SELECT bot_user_id FROM bot_users WHERE id = $1 AND owner_user_id = $2`, id, ownerUserID).Scan(&botUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var configName string
	err = tx.QueryRow(`SELECT name FROM llm_configs WHERE id = $1 AND (owner_user_id = $2 OR shared = TRUE)`, llmConfigID, ownerUserID).Scan(&configName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if _, err = tx.Exec(`UPDATE users SET username = $1, bio = $2 WHERE id = $3`, name, description, botUserID); err != nil {
		return nil, err
	}
	var item BotUser
	err = tx.QueryRow(
		`UPDATE bot_users
		    SET name = $3, description = $4, system_prompt = $5, llm_config_id = $6, updated_at = $7
		  WHERE id = $1 AND owner_user_id = $2
		  RETURNING id, owner_user_id, bot_user_id, name, description, system_prompt, llm_config_id, created_at, updated_at`,
		id, ownerUserID, name, description, systemPrompt, llmConfigID, now,
	).Scan(&item.ID, &item.OwnerUserID, &item.BotUserID, &item.Name, &item.Description, &item.SystemPrompt, &item.LLMConfigID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item.ConfigName = configName
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Server) deleteBotUser(ownerUserID string, id int64) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	var botUserID string
	err = tx.QueryRow(`SELECT bot_user_id FROM bot_users WHERE id = $1 AND owner_user_id = $2`, id, ownerUserID).Scan(&botUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if _, err = tx.Exec(`DELETE FROM bot_users WHERE id = $1 AND owner_user_id = $2`, id, ownerUserID); err != nil {
		return false, err
	}
	if _, err = tx.Exec(`DELETE FROM users WHERE id = $1`, botUserID); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Server) getBotUserByUserID(botUserID string) (*BotUser, error) {
	var item BotUser
	err := s.db.QueryRow(
		`SELECT b.id, b.owner_user_id, b.bot_user_id, b.name, b.description, b.system_prompt, b.llm_config_id, c.name, b.created_at, b.updated_at
		   FROM bot_users b
		   JOIN llm_configs c ON c.id = b.llm_config_id
		  WHERE b.bot_user_id = $1`,
		botUserID,
	).Scan(&item.ID, &item.OwnerUserID, &item.BotUserID, &item.Name, &item.Description, &item.SystemPrompt, &item.LLMConfigID, &item.ConfigName, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *Server) getLLMConfigForThread(chatThreadID, llmThreadID int64, botUserID string) (*LLMConfig, string, error) {
	var item LLMConfig
	var apiKey string
	err := s.db.QueryRow(
		`SELECT c.id, c.owner_user_id, c.name, c.base_url, c.model, c.api_key, c.system_prompt, (c.api_key <> '') AS has_api_key, c.created_at, c.updated_at
		   FROM llm_threads t
		   LEFT JOIN bot_users b ON b.bot_user_id = t.bot_user_id
		   JOIN llm_configs c ON c.id = COALESCE(t.llm_config_id, b.llm_config_id)
		  WHERE t.id = $1 AND t.chat_thread_id = $2 AND t.bot_user_id = $3`,
		llmThreadID,
		chatThreadID,
		botUserID,
	).Scan(&item.ID, &item.OwnerUserID, &item.Name, &item.BaseURL, &item.Model, &apiKey, &item.SystemPrompt, &item.HasAPIKey, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", err
	}
	return &item, apiKey, nil
}

func (s *Server) createUser(user *User) error {
	if user.Role == "" {
		user.Role = "user"
	}
	if user.IconURL == "" {
		user.IconURL = ""
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`LOCK TABLE users IN EXCLUSIVE MODE`); err != nil {
		return err
	}

	role := user.Role
	if role == "user" {
		var realUserCount int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM users WHERE id <> $1`, systemUserID).Scan(&realUserCount); err != nil {
			return err
		}
		if realUserCount == 0 {
			role = "admin"
		}
	}

	_, err = tx.Exec(
		`INSERT INTO users (id, username, email, password_hash, role, bio, icon_url, is_online, last_active_device_type, last_seen_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		user.ID,
		user.Username,
		user.Email,
		user.Password,
		role,
		user.Bio,
		user.IconURL,
		user.IsOnline,
		normalizeDeviceType(user.DeviceType),
		user.LastSeenAt,
		user.CreatedAt,
	)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return errEmailExists
		}
		return err
	}

	user.Role = role
	return tx.Commit()
}

func (s *Server) upsertUserDevice(userID, deviceType, pushToken string, loginAt time.Time) error {
	deviceType = normalizeDeviceType(deviceType)
	deviceID := normalizeDeviceID("", deviceType)
	pushToken = sanitizePushToken(pushToken)
	_, err := s.db.Exec(
		`INSERT INTO user_devices (user_id, device_type, device_id, push_token, push_enabled, is_online, last_login_at, last_seen_at, last_active_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, TRUE, FALSE, $5, $5, $5, $5, $5)
		 ON CONFLICT (user_id, device_type, device_id)
		 DO UPDATE SET push_token = EXCLUDED.push_token,
		               push_enabled = CASE WHEN EXCLUDED.push_token <> '' THEN TRUE ELSE user_devices.push_enabled END,
		               last_login_at = EXCLUDED.last_login_at,
		               last_seen_at = EXCLUDED.last_seen_at,
		               last_active_at = EXCLUDED.last_active_at,
		               updated_at = EXCLUDED.updated_at`,
		userID,
		deviceType,
		deviceID,
		pushToken,
		loginAt,
	)
	return err
}

func (s *Server) upsertUserDeviceWithID(userID, deviceType, deviceID, pushToken string, loginAt time.Time) error {
	deviceType = normalizeDeviceType(deviceType)
	deviceID = normalizeDeviceID(deviceID, deviceType)
	pushToken = sanitizePushToken(pushToken)
	_, err := s.db.Exec(
		`INSERT INTO user_devices (user_id, device_type, device_id, push_token, push_enabled, is_online, last_login_at, last_seen_at, last_active_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, CASE WHEN $4 <> '' THEN TRUE ELSE TRUE END, FALSE, $5, $5, $5, $5, $5)
		 ON CONFLICT (user_id, device_type, device_id)
		 DO UPDATE SET push_token = CASE WHEN EXCLUDED.push_token <> '' THEN EXCLUDED.push_token ELSE user_devices.push_token END,
		               push_enabled = CASE WHEN EXCLUDED.push_token <> '' THEN TRUE ELSE user_devices.push_enabled END,
		               last_login_at = EXCLUDED.last_login_at,
		               last_seen_at = EXCLUDED.last_seen_at,
		               last_active_at = EXCLUDED.last_active_at,
		               updated_at = EXCLUDED.updated_at`,
		userID,
		deviceType,
		deviceID,
		pushToken,
		loginAt,
	)
	return err
}

func (s *Server) updateUserDevicePresence(userID, deviceType, deviceID string, isOnline bool, seenAt time.Time) error {
	deviceType = normalizeDeviceType(deviceType)
	deviceID = normalizeDeviceID(deviceID, deviceType)
	_, err := s.db.Exec(
		`INSERT INTO user_devices (user_id, device_type, device_id, push_token, push_enabled, is_online, last_login_at, last_seen_at, last_active_at, created_at, updated_at)
		 VALUES ($1, $2, $3, '', TRUE, $4, $5, $5, $5, $5, $5)
		 ON CONFLICT (user_id, device_type, device_id)
		 DO UPDATE SET is_online = EXCLUDED.is_online,
		               last_seen_at = EXCLUDED.last_seen_at,
		               last_active_at = EXCLUDED.last_active_at,
		               updated_at = EXCLUDED.updated_at`,
		userID,
		deviceType,
		deviceID,
		isOnline,
		seenAt,
	)
	return err
}

func (s *Server) updateUserDevicePushToken(userID, deviceType, deviceID, pushToken string, now time.Time) error {
	deviceType = normalizeDeviceType(deviceType)
	deviceID = normalizeDeviceID(deviceID, deviceType)
	pushToken = sanitizePushToken(pushToken)
	_, err := s.db.Exec(
		`INSERT INTO user_devices (user_id, device_type, device_id, push_token, push_enabled, is_online, last_login_at, last_seen_at, last_active_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, CASE WHEN $4 <> '' THEN TRUE ELSE FALSE END, FALSE, $5, $5, $5, $5, $5)
		 ON CONFLICT (user_id, device_type, device_id)
		 DO UPDATE SET push_token = EXCLUDED.push_token,
		               push_enabled = CASE WHEN EXCLUDED.push_token <> '' THEN TRUE ELSE FALSE END,
		               last_seen_at = EXCLUDED.last_seen_at,
		               last_active_at = EXCLUDED.last_active_at,
		               updated_at = EXCLUDED.updated_at`,
		userID,
		deviceType,
		deviceID,
		pushToken,
		now,
	)
	return err
}

func (s *Server) clearUserDevicePushToken(userID, deviceID string, now time.Time) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.New("device id required")
	}
	_, err := s.db.Exec(
		`UPDATE user_devices
		    SET push_token = '',
		        push_enabled = FALSE,
		        last_seen_at = $3,
		        last_active_at = $3,
		        updated_at = $3
		  WHERE user_id = $1 AND device_id = $2`,
		userID,
		deviceID,
		now,
	)
	return err
}

func (s *Server) listUserDevices(userID string) ([]UserDevice, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, device_type, device_id, push_token, push_enabled, app_version, is_online, last_login_at, last_seen_at, last_active_at, created_at, updated_at
		   FROM user_devices
		  WHERE user_id = $1
		  ORDER BY updated_at DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]UserDevice, 0)
	for rows.Next() {
		var item UserDevice
		var lastSeenAt sql.NullTime
		var lastActiveAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.DeviceType,
			&item.DeviceID,
			&item.PushToken,
			&item.PushEnabled,
			&item.AppVersion,
			&item.IsOnline,
			&item.LastLoginAt,
			&lastSeenAt,
			&lastActiveAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if lastSeenAt.Valid {
			item.LastSeenAt = &lastSeenAt.Time
		}
		if lastActiveAt.Valid {
			item.LastActiveAt = &lastActiveAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) listPushableUserDevices(userID string) ([]UserDevice, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, device_type, device_id, push_token, push_enabled, app_version, is_online, last_login_at, last_seen_at, last_active_at, created_at, updated_at
		   FROM user_devices
		  WHERE user_id = $1
		    AND push_enabled = TRUE
		    AND push_token <> ''
		  ORDER BY updated_at DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]UserDevice, 0)
	for rows.Next() {
		var item UserDevice
		var lastSeenAt sql.NullTime
		var lastActiveAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.DeviceType,
			&item.DeviceID,
			&item.PushToken,
			&item.PushEnabled,
			&item.AppVersion,
			&item.IsOnline,
			&item.LastLoginAt,
			&lastSeenAt,
			&lastActiveAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if lastSeenAt.Valid {
			item.LastSeenAt = &lastSeenAt.Time
		}
		if lastActiveAt.Valid {
			item.LastActiveAt = &lastActiveAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) createMarkdownEntry(userID, title, filePath string, isPublic bool, uploadedAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO markdown_entries (user_id, title, file_path, is_public, uploaded_at) VALUES ($1, $2, $3, $4, $5)`,
		userID,
		title,
		filePath,
		isPublic,
		uploadedAt,
	)
	return err
}

func (s *Server) createMarkdownEntryReturningID(userID, title, filePath string, isPublic bool, uploadedAt time.Time) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO markdown_entries (user_id, title, file_path, is_public, uploaded_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		userID,
		title,
		filePath,
		isPublic,
		uploadedAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Server) getMarkdownEntryByID(id int64) (*MarkdownEntry, error) {
	var entry MarkdownEntry
	err := s.db.QueryRow(
		`SELECT id, user_id, title, file_path, is_public, uploaded_at
		   FROM markdown_entries
		  WHERE id = $1`,
		id,
	).Scan(&entry.ID, &entry.UserID, &entry.Title, &entry.FilePath, &entry.IsPublic, &entry.UploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
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

func (s *Server) listWebAuthnCredentialSummaries(userID string) ([]WebAuthnCredentialSummary, error) {
	rows, err := s.db.Query(
		`SELECT credential_id, created_at, updated_at
		   FROM webauthn_credentials
		  WHERE user_id = $1
		  ORDER BY updated_at DESC, created_at DESC, credential_id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]WebAuthnCredentialSummary, 0)
	for rows.Next() {
		var item WebAuthnCredentialSummary
		if err := rows.Scan(&item.CredentialID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
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

func (s *Server) deleteWebAuthnCredential(userID, credentialID string) (bool, error) {
	result, err := s.db.Exec(
		`DELETE FROM webauthn_credentials
		  WHERE credential_id = $1 AND user_id = $2`,
		credentialID,
		userID,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *Server) updateUserIcon(userID, iconURL string) error {
	_, err := s.db.Exec(
		`UPDATE users SET icon_url = $1 WHERE id = $2`,
		iconURL,
		userID,
	)
	return err
}

func (s *Server) updateUserBio(userID, bio string) error {
	_, err := s.db.Exec(
		`UPDATE users SET bio = $1 WHERE id = $2`,
		bio,
		userID,
	)
	return err
}

func (s *Server) upsertProfileRecommendation(targetUserID, authorUserID, content string, now time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO profile_recommendations (target_user_id, author_user_id, content, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $4)
		 ON CONFLICT (target_user_id, author_user_id)
		 DO UPDATE SET content = EXCLUDED.content, updated_at = EXCLUDED.updated_at`,
		targetUserID,
		authorUserID,
		content,
		now,
	)
	return err
}

func (s *Server) blockUser(blockerUserID, blockedUserID string, now time.Time) error {
	if blockerUserID == blockedUserID {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO user_blocks (blocker_user_id, blocked_user_id, created_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (blocker_user_id, blocked_user_id) DO NOTHING`,
		blockerUserID,
		blockedUserID,
		now,
	)
	return err
}

func (s *Server) unblockUser(blockerUserID, blockedUserID string) (bool, error) {
	result, err := s.db.Exec(
		`DELETE FROM user_blocks
		  WHERE blocker_user_id = $1 AND blocked_user_id = $2`,
		blockerUserID,
		blockedUserID,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *Server) getUserBlockState(viewerUserID, targetUserID string) (bool, bool, error) {
	if strings.TrimSpace(viewerUserID) == "" || strings.TrimSpace(targetUserID) == "" || viewerUserID == targetUserID {
		return false, false, nil
	}
	var iBlockedUser bool
	var blockedMe bool
	err := s.db.QueryRow(
		`SELECT EXISTS(
		     SELECT 1 FROM user_blocks
		      WHERE blocker_user_id = $1 AND blocked_user_id = $2
		   ),
		   EXISTS(
		     SELECT 1 FROM user_blocks
		      WHERE blocker_user_id = $2 AND blocked_user_id = $1
		   )`,
		viewerUserID,
		targetUserID,
	).Scan(&iBlockedUser, &blockedMe)
	if err != nil {
		return false, false, err
	}
	return iBlockedUser, blockedMe, nil
}

func (s *Server) listProfileRecommendations(targetUserID string) ([]ProfileRecommendation, error) {
	rows, err := s.db.Query(
		`SELECT pr.id, pr.target_user_id, pr.author_user_id, u.username, u.icon_url, pr.content, pr.created_at, pr.updated_at
		   FROM profile_recommendations pr
		   JOIN users u ON u.id = pr.author_user_id
		  WHERE pr.target_user_id = $1
		  ORDER BY pr.updated_at DESC, pr.id DESC`,
		targetUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ProfileRecommendation, 0)
	for rows.Next() {
		var item ProfileRecommendation
		if err := rows.Scan(
			&item.ID,
			&item.TargetUserID,
			&item.AuthorUserID,
			&item.AuthorUsername,
			&item.AuthorUserIcon,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) getUserProfileDetail(targetUserID, viewerUserID string) (*UserProfileDetail, error) {
	user, err := s.getUserByID(targetUserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	recommendations, err := s.listProfileRecommendations(targetUserID)
	if err != nil {
		return nil, err
	}
	iBlockedUser, blockedMe, err := s.getUserBlockState(viewerUserID, targetUserID)
	if err != nil {
		return nil, err
	}
	return &UserProfileDetail{
		UserID:   user.ID,
		Username: user.Username,
		Email: func() string {
			if targetUserID == viewerUserID {
				return user.Email
			}
			return ""
		}(),
		IconURL:         user.IconURL,
		Bio:             user.Bio,
		CreatedAt:       user.CreatedAt,
		IsMe:            targetUserID == viewerUserID,
		CanRecommend:    targetUserID != viewerUserID,
		IBlockedUser:    iBlockedUser,
		BlockedMe:       blockedMe,
		Recommendations: recommendations,
	}, nil
}

func (s *Server) listMarkdownEntries(userID string, limit, offset int) ([]MarkdownEntry, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(
		`SELECT id, user_id, title, file_path, is_public, uploaded_at
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
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Title, &entry.FilePath, &entry.IsPublic, &entry.UploadedAt); err != nil {
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

func (s *Server) listPublicMarkdownEntries(limit, offset int) ([]PublicMarkdownEntry, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.db.Query(
		`SELECT m.id, m.user_id, u.username, u.icon_url, m.title, m.uploaded_at
		   FROM markdown_entries m
		   JOIN users u ON u.id = m.user_id
		  WHERE m.is_public = TRUE
		  ORDER BY m.uploaded_at DESC
		  LIMIT $1 OFFSET $2`,
		limit+1,
		offset,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	entries := make([]PublicMarkdownEntry, 0, limit+1)
	for rows.Next() {
		var entry PublicMarkdownEntry
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Username, &entry.UserIcon, &entry.Title, &entry.UploadedAt); err != nil {
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

func (s *Server) getSiteSettings() (*SiteSettings, error) {
	var settings SiteSettings
	var devURL sql.NullString
	var devName sql.NullString
	var devUploadedAt sql.NullTime
	var prodURL sql.NullString
	var prodName sql.NullString
	var prodUploadedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT name, description, icon_url,
		        apple_push_dev_cert_url, apple_push_dev_cert_name, apple_push_dev_cert_uploaded_at,
		        apple_push_prod_cert_url, apple_push_prod_cert_name, apple_push_prod_cert_uploaded_at,
		        updated_at
		 FROM site_settings
		 WHERE id = 1`,
	).Scan(
		&settings.Name,
		&settings.Description,
		&settings.IconURL,
		&devURL,
		&devName,
		&devUploadedAt,
		&prodURL,
		&prodName,
		&prodUploadedAt,
		&settings.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &SiteSettings{
				Name:        "Polar-",
				Description: "",
				IconURL:     "",
				UpdatedAt:   time.Now(),
			}, nil
		}
		return nil, err
	}
	if devURL.Valid && devURL.String != "" {
		settings.ApplePushDevCert = &ApplePushCertificate{
			Environment: "dev",
			FileName:    devName.String,
			FileURL:     devURL.String,
		}
		if devUploadedAt.Valid {
			settings.ApplePushDevCert.UploadedAt = &devUploadedAt.Time
		}
	}
	if prodURL.Valid && prodURL.String != "" {
		settings.ApplePushProdCert = &ApplePushCertificate{
			Environment: "prod",
			FileName:    prodName.String,
			FileURL:     prodURL.String,
		}
		if prodUploadedAt.Valid {
			settings.ApplePushProdCert.UploadedAt = &prodUploadedAt.Time
		}
	}
	return &settings, nil
}

func (s *Server) updateSiteSettings(name, description string) error {
	_, err := s.db.Exec(
		`INSERT INTO site_settings (
		     id, name, description, icon_url,
		     apple_push_dev_cert_url, apple_push_dev_cert_name, apple_push_dev_cert_uploaded_at,
		     apple_push_prod_cert_url, apple_push_prod_cert_name, apple_push_prod_cert_uploaded_at,
		     updated_at
		 )
		 VALUES (
		     1, $1, $2, COALESCE((SELECT icon_url FROM site_settings WHERE id = 1), ''),
		     COALESCE((SELECT apple_push_dev_cert_url FROM site_settings WHERE id = 1), ''),
		     COALESCE((SELECT apple_push_dev_cert_name FROM site_settings WHERE id = 1), ''),
		     (SELECT apple_push_dev_cert_uploaded_at FROM site_settings WHERE id = 1),
		     COALESCE((SELECT apple_push_prod_cert_url FROM site_settings WHERE id = 1), ''),
		     COALESCE((SELECT apple_push_prod_cert_name FROM site_settings WHERE id = 1), ''),
		     (SELECT apple_push_prod_cert_uploaded_at FROM site_settings WHERE id = 1),
		     NOW()
		 )
		 ON CONFLICT (id)
		 DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW()`,
		name,
		description,
	)
	return err
}

func (s *Server) updateSiteIcon(iconURL string) error {
	_, err := s.db.Exec(
		`INSERT INTO site_settings (
		     id, name, description, icon_url,
		     apple_push_dev_cert_url, apple_push_dev_cert_name, apple_push_dev_cert_uploaded_at,
		     apple_push_prod_cert_url, apple_push_prod_cert_name, apple_push_prod_cert_uploaded_at,
		     updated_at
		 )
		 VALUES (1, COALESCE((SELECT name FROM site_settings WHERE id = 1), 'Polar-'),
		             COALESCE((SELECT description FROM site_settings WHERE id = 1), ''),
		             $1,
		             COALESCE((SELECT apple_push_dev_cert_url FROM site_settings WHERE id = 1), ''),
		             COALESCE((SELECT apple_push_dev_cert_name FROM site_settings WHERE id = 1), ''),
		             (SELECT apple_push_dev_cert_uploaded_at FROM site_settings WHERE id = 1),
		             COALESCE((SELECT apple_push_prod_cert_url FROM site_settings WHERE id = 1), ''),
		             COALESCE((SELECT apple_push_prod_cert_name FROM site_settings WHERE id = 1), ''),
		             (SELECT apple_push_prod_cert_uploaded_at FROM site_settings WHERE id = 1),
		             NOW())
		 ON CONFLICT (id)
		 DO UPDATE SET icon_url = EXCLUDED.icon_url, updated_at = NOW()`,
		iconURL,
	)
	return err
}

func (s *Server) updateApplePushCertificate(environment, fileURL, fileName string, uploadedAt time.Time) error {
	query := ""
	switch environment {
	case "dev":
		query = `INSERT INTO site_settings (
		           id, name, description, icon_url,
		           apple_push_dev_cert_url, apple_push_dev_cert_name, apple_push_dev_cert_uploaded_at,
		           apple_push_prod_cert_url, apple_push_prod_cert_name, apple_push_prod_cert_uploaded_at,
		           updated_at
		         )
		         VALUES (
		           1,
		           COALESCE((SELECT name FROM site_settings WHERE id = 1), 'Polar-'),
		           COALESCE((SELECT description FROM site_settings WHERE id = 1), ''),
		           COALESCE((SELECT icon_url FROM site_settings WHERE id = 1), ''),
		           $1, $2, $3,
		           COALESCE((SELECT apple_push_prod_cert_url FROM site_settings WHERE id = 1), ''),
		           COALESCE((SELECT apple_push_prod_cert_name FROM site_settings WHERE id = 1), ''),
		           (SELECT apple_push_prod_cert_uploaded_at FROM site_settings WHERE id = 1),
		           NOW()
		         )
		         ON CONFLICT (id)
		         DO UPDATE SET apple_push_dev_cert_url = EXCLUDED.apple_push_dev_cert_url,
		                       apple_push_dev_cert_name = EXCLUDED.apple_push_dev_cert_name,
		                       apple_push_dev_cert_uploaded_at = EXCLUDED.apple_push_dev_cert_uploaded_at,
		                       updated_at = NOW()`
	case "prod":
		query = `INSERT INTO site_settings (
		           id, name, description, icon_url,
		           apple_push_dev_cert_url, apple_push_dev_cert_name, apple_push_dev_cert_uploaded_at,
		           apple_push_prod_cert_url, apple_push_prod_cert_name, apple_push_prod_cert_uploaded_at,
		           updated_at
		         )
		         VALUES (
		           1,
		           COALESCE((SELECT name FROM site_settings WHERE id = 1), 'Polar-'),
		           COALESCE((SELECT description FROM site_settings WHERE id = 1), ''),
		           COALESCE((SELECT icon_url FROM site_settings WHERE id = 1), ''),
		           COALESCE((SELECT apple_push_dev_cert_url FROM site_settings WHERE id = 1), ''),
		           COALESCE((SELECT apple_push_dev_cert_name FROM site_settings WHERE id = 1), ''),
		           (SELECT apple_push_dev_cert_uploaded_at FROM site_settings WHERE id = 1),
		           $1, $2, $3,
		           NOW()
		         )
		         ON CONFLICT (id)
		         DO UPDATE SET apple_push_prod_cert_url = EXCLUDED.apple_push_prod_cert_url,
		                       apple_push_prod_cert_name = EXCLUDED.apple_push_prod_cert_name,
		                       apple_push_prod_cert_uploaded_at = EXCLUDED.apple_push_prod_cert_uploaded_at,
		                       updated_at = NOW()`
	default:
		return errors.New("invalid apple push certificate environment")
	}

	_, err := s.db.Exec(query, fileURL, fileName, uploadedAt)
	return err
}

func (s *Server) clearApplePushCertificate(environment string) error {
	query := ""
	switch environment {
	case "dev":
		query = `UPDATE site_settings
		            SET apple_push_dev_cert_url = '',
		                apple_push_dev_cert_name = '',
		                apple_push_dev_cert_uploaded_at = NULL,
		                updated_at = NOW()
		          WHERE id = 1`
	case "prod":
		query = `UPDATE site_settings
		            SET apple_push_prod_cert_url = '',
		                apple_push_prod_cert_name = '',
		                apple_push_prod_cert_uploaded_at = NULL,
		                updated_at = NOW()
		          WHERE id = 1`
	default:
		return errors.New("invalid apple push certificate environment")
	}
	_, err := s.db.Exec(query)
	return err
}

func (s *Server) getMarkdownEntryForUser(viewerUserID string, id int64) (*MarkdownEntry, bool, error) {
	var entry MarkdownEntry
	err := s.db.QueryRow(
		`SELECT id, user_id, title, file_path, is_public, uploaded_at
		FROM markdown_entries
		WHERE id = $1 AND (user_id = $2 OR is_public = TRUE)`,
		id,
		viewerUserID,
	).Scan(&entry.ID, &entry.UserID, &entry.Title, &entry.FilePath, &entry.IsPublic, &entry.UploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &entry, entry.UserID == viewerUserID, nil
}

func (s *Server) getOwnedMarkdownEntry(userID string, id int64) (*MarkdownEntry, error) {
	var entry MarkdownEntry
	err := s.db.QueryRow(
		`SELECT id, user_id, title, file_path, is_public, uploaded_at
		FROM markdown_entries
		WHERE user_id = $1 AND id = $2`,
		userID,
		id,
	).Scan(&entry.ID, &entry.UserID, &entry.Title, &entry.FilePath, &entry.IsPublic, &entry.UploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

func (s *Server) updateMarkdownEntry(userID string, id int64, title, filePath string, isPublic bool) error {
	_, err := s.db.Exec(
		`UPDATE markdown_entries SET title = $1, file_path = $2, is_public = $3 WHERE user_id = $4 AND id = $5`,
		title,
		filePath,
		isPublic,
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

func (s *Server) createPost(userID string, tagID *int64, postType, content string, createdAt time.Time) (int64, error) {
	if postType == "" {
		postType = "standard"
	}
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO posts (user_id, tag_id, post_type, content, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		userID,
		tagID,
		postType,
		content,
		createdAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Server) deletePost(postID int64) (bool, error) {
	result, err := s.db.Exec(`DELETE FROM posts WHERE id = $1`, postID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (s *Server) addPostImage(postID int64, imageItem PostImage, createdAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO post_images (post_id, file_url, small_url, medium_url, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		postID,
		imageItem.OriginalURL,
		imageItem.SmallURL,
		imageItem.MediumURL,
		createdAt,
	)
	return err
}

func (s *Server) addPostVideo(postID int64, fileURL, posterURL string, createdAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO post_videos (post_id, file_url, poster_url, created_at)
		 VALUES ($1, $2, $3, $4)`,
		postID,
		fileURL,
		posterURL,
		createdAt,
	)
	return err
}

func normalizePostImageItem(originalURL, smallURL, mediumURL string) PostImage {
	item := PostImage{
		OriginalURL: originalURL,
		SmallURL:    smallURL,
		MediumURL:   mediumURL,
	}
	if item.OriginalURL == "" {
		if item.MediumURL != "" {
			item.OriginalURL = item.MediumURL
		} else {
			item.OriginalURL = item.SmallURL
		}
	}
	if item.MediumURL == "" {
		item.MediumURL = item.OriginalURL
	}
	if item.SmallURL == "" {
		item.SmallURL = item.MediumURL
	}
	return item
}

func legacyPostImageURL(item PostImage) string {
	if item.MediumURL != "" {
		return item.MediumURL
	}
	if item.OriginalURL != "" {
		return item.OriginalURL
	}
	return item.SmallURL
}

func (s *Server) listPosts(userID string, limit, offset int, filterTagID *int64, filterPostType string) ([]Post, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	if filterPostType == "" {
		filterPostType = "all"
	}
	rows, err := s.db.Query(
		`SELECT p.id, p.user_id, u.username, u.icon_url, p.tag_id, p.post_type, p.content, p.created_at,
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
		  WHERE ($2::BIGINT IS NULL OR p.tag_id = $2)
		    AND ($3 = 'all' OR p.post_type = $3)
		  ORDER BY p.created_at DESC
		  LIMIT $4 OFFSET $5`,
		userID,
		filterTagID,
		filterPostType,
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
			&post.UserIcon,
			&post.TagID,
			&post.PostType,
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
		`SELECT post_id, file_url, small_url, medium_url FROM post_images
		  WHERE post_id = ANY($1)
		  ORDER BY id ASC`,
		pq.Array(postIDs),
	)
	if err != nil {
		return posts, hasMore, err
	}
	defer imageRows.Close()

	imageMap := make(map[int64][]string, len(postIDs))
	imageItemMap := make(map[int64][]PostImage, len(postIDs))
	for imageRows.Next() {
		var postID int64
		var fileURL string
		var smallURL string
		var mediumURL string
		if err := imageRows.Scan(&postID, &fileURL, &smallURL, &mediumURL); err != nil {
			return posts, hasMore, err
		}
		imageItem := normalizePostImageItem(fileURL, smallURL, mediumURL)
		imageMap[postID] = append(imageMap[postID], legacyPostImageURL(imageItem))
		imageItemMap[postID] = append(imageItemMap[postID], imageItem)
	}
	if err := imageRows.Err(); err != nil {
		return posts, hasMore, err
	}

	videoRows, err := s.db.Query(
		`SELECT post_id, file_url, poster_url FROM post_videos
		  WHERE post_id = ANY($1)
		  ORDER BY id ASC`,
		pq.Array(postIDs),
	)
	if err != nil {
		return posts, hasMore, err
	}
	defer videoRows.Close()

	videoMap := make(map[int64][]string, len(postIDs))
	videoItemMap := make(map[int64][]PostVideo, len(postIDs))
	for videoRows.Next() {
		var postID int64
		var fileURL string
		var posterURL string
		if err := videoRows.Scan(&postID, &fileURL, &posterURL); err != nil {
			return posts, hasMore, err
		}
		videoMap[postID] = append(videoMap[postID], fileURL)
		videoItemMap[postID] = append(videoItemMap[postID], PostVideo{URL: fileURL, PosterURL: posterURL})
	}
	if err := videoRows.Err(); err != nil {
		return posts, hasMore, err
	}

	for i := range posts {
		posts[i].Images = imageMap[posts[i].ID]
		posts[i].ImageItems = imageItemMap[posts[i].ID]
		posts[i].Videos = videoMap[posts[i].ID]
		posts[i].VideoItems = videoItemMap[posts[i].ID]
	}

	if err := s.attachTaskData(posts, userID); err != nil {
		return posts, hasMore, err
	}

	return posts, hasMore, nil
}

func (s *Server) getPostByID(userID string, postID int64) (*Post, error) {
	var post Post
	err := s.db.QueryRow(
		`SELECT p.id, p.user_id, u.username, u.icon_url, p.tag_id, p.post_type, p.content, p.created_at,
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
		&post.UserIcon,
		&post.TagID,
		&post.PostType,
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
		`SELECT file_url, small_url, medium_url FROM post_images WHERE post_id = $1 ORDER BY id ASC`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []string
	var imageItems []PostImage
	for rows.Next() {
		var url string
		var smallURL string
		var mediumURL string
		if err := rows.Scan(&url, &smallURL, &mediumURL); err != nil {
			return nil, err
		}
		imageItem := normalizePostImageItem(url, smallURL, mediumURL)
		images = append(images, legacyPostImageURL(imageItem))
		imageItems = append(imageItems, imageItem)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	post.Images = images
	post.ImageItems = imageItems

	videoRows, err := s.db.Query(
		`SELECT file_url, poster_url FROM post_videos WHERE post_id = $1 ORDER BY id ASC`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer videoRows.Close()

	var videos []string
	var videoItems []PostVideo
	for videoRows.Next() {
		var url string
		var posterURL string
		if err := videoRows.Scan(&url, &posterURL); err != nil {
			return nil, err
		}
		videos = append(videos, url)
		videoItems = append(videoItems, PostVideo{URL: url, PosterURL: posterURL})
	}
	if err := videoRows.Err(); err != nil {
		return nil, err
	}
	post.Videos = videos
	post.VideoItems = videoItems

	if post.PostType == "task" {
		task, _, err := s.getTaskPostByID(post.ID, userID)
		if err != nil {
			return nil, err
		}
		post.Task = task
	}

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
		`SELECT r.id, r.post_id, r.user_id, u.username, u.icon_url, r.content, r.created_at
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
			&reply.UserIcon,
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

func (s *Server) createTaskPost(postID int64, location string, startAt, endAt time.Time, workingHours string, applyDeadline time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO task_posts (post_id, location, start_at, end_at, working_hours, apply_deadline, application_status)
		 VALUES ($1, $2, $3, $4, $5, $6, 'open')`,
		postID,
		location,
		startAt,
		endAt,
		workingHours,
		applyDeadline,
	)
	return err
}

func (s *Server) attachTaskData(posts []Post, currentUserID string) error {
	postIDs := make([]int64, 0, len(posts))
	taskIndex := make(map[int64]*Post)
	for i := range posts {
		if posts[i].PostType != "task" {
			continue
		}
		postIDs = append(postIDs, posts[i].ID)
		taskIndex[posts[i].ID] = &posts[i]
	}
	if len(postIDs) == 0 {
		return nil
	}

	rows, err := s.db.Query(
		`SELECT tp.post_id, tp.location, tp.start_at, tp.end_at, tp.working_hours, tp.apply_deadline,
		        tp.application_status, tp.selected_applicant_id, sa.username, tp.selected_at,
		        tp.invitation_template, tp.invitation_sent_at,
		        COALESCE(ac.applicant_count, 0) AS applicant_count,
		        EXISTS(
		          SELECT 1 FROM task_applications ta
		           WHERE ta.post_id = tp.post_id AND ta.user_id = $2 AND ta.withdrawn_at IS NULL
		        ) AS applied_by_me
		   FROM task_posts tp
		   LEFT JOIN users sa ON sa.id = tp.selected_applicant_id
		   LEFT JOIN (
		     SELECT post_id, COUNT(*) AS applicant_count
		       FROM task_applications
		      WHERE withdrawn_at IS NULL
		        AND post_id = ANY($1)
		      GROUP BY post_id
		   ) ac ON ac.post_id = tp.post_id
		  WHERE tp.post_id = ANY($1)`,
		pq.Array(postIDs),
		currentUserID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var (
			postID                int64
			task                  TaskPost
			selectedApplicantID   sql.NullString
			selectedApplicantName sql.NullString
			selectedAt            sql.NullTime
			invitationSentAt      sql.NullTime
		)
		if err := rows.Scan(
			&postID,
			&task.Location,
			&task.StartAt,
			&task.EndAt,
			&task.WorkingHours,
			&task.ApplyDeadline,
			&task.ApplicationStatus,
			&selectedApplicantID,
			&selectedApplicantName,
			&selectedAt,
			&task.InvitationTemplate,
			&invitationSentAt,
			&task.ApplicantCount,
			&task.AppliedByMe,
		); err != nil {
			return err
		}
		task.PostID = postID
		if selectedApplicantID.Valid {
			task.SelectedApplicantID = selectedApplicantID.String
		}
		if selectedApplicantName.Valid {
			task.SelectedApplicantName = selectedApplicantName.String
		}
		if selectedAt.Valid {
			task.SelectedAt = &selectedAt.Time
		}
		if invitationSentAt.Valid {
			task.InvitationSentAt = &invitationSentAt.Time
		}
		if post := taskIndex[postID]; post != nil {
			task.SelectedByMe = task.SelectedApplicantID != "" && task.SelectedApplicantID == currentUserID
			task.CanManage = post.UserID == currentUserID
			task.CanApply = post.UserID != currentUserID &&
				task.SelectedApplicantID == "" &&
				(task.AppliedByMe || (task.ApplicationStatus == "open" && now.Before(task.ApplyDeadline)))
			task.CanViewResults = task.CanManage || task.SelectedByMe
			task.CanSubmitResult = task.SelectedByMe
			post.Task = &task
		}
	}
	return rows.Err()
}

func (s *Server) getTaskPostByID(postID int64, currentUserID string) (*TaskPost, string, error) {
	var (
		postOwner             string
		task                  TaskPost
		selectedApplicantID   sql.NullString
		selectedApplicantName sql.NullString
		selectedAt            sql.NullTime
		invitationSentAt      sql.NullTime
	)
	err := s.db.QueryRow(
		`SELECT p.user_id, tp.post_id, tp.location, tp.start_at, tp.end_at, tp.working_hours, tp.apply_deadline,
		        tp.application_status, tp.selected_applicant_id, sa.username, tp.selected_at,
		        tp.invitation_template, tp.invitation_sent_at,
		        COALESCE(ac.applicant_count, 0) AS applicant_count,
		        EXISTS(
		          SELECT 1 FROM task_applications ta
		           WHERE ta.post_id = tp.post_id AND ta.user_id = $2 AND ta.withdrawn_at IS NULL
		        ) AS applied_by_me
		   FROM task_posts tp
		   JOIN posts p ON p.id = tp.post_id
		   LEFT JOIN users sa ON sa.id = tp.selected_applicant_id
		   LEFT JOIN (
		     SELECT post_id, COUNT(*) AS applicant_count
		       FROM task_applications
		      WHERE withdrawn_at IS NULL
		      GROUP BY post_id
		   ) ac ON ac.post_id = tp.post_id
		  WHERE tp.post_id = $1`,
		postID,
		currentUserID,
	).Scan(
		&postOwner,
		&task.PostID,
		&task.Location,
		&task.StartAt,
		&task.EndAt,
		&task.WorkingHours,
		&task.ApplyDeadline,
		&task.ApplicationStatus,
		&selectedApplicantID,
		&selectedApplicantName,
		&selectedAt,
		&task.InvitationTemplate,
		&invitationSentAt,
		&task.ApplicantCount,
		&task.AppliedByMe,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", err
	}
	if selectedApplicantID.Valid {
		task.SelectedApplicantID = selectedApplicantID.String
	}
	if selectedApplicantName.Valid {
		task.SelectedApplicantName = selectedApplicantName.String
	}
	if selectedAt.Valid {
		task.SelectedAt = &selectedAt.Time
	}
	if invitationSentAt.Valid {
		task.InvitationSentAt = &invitationSentAt.Time
	}
	now := time.Now()
	task.SelectedByMe = task.SelectedApplicantID != "" && task.SelectedApplicantID == currentUserID
	task.CanManage = postOwner == currentUserID
	task.CanApply = postOwner != currentUserID &&
		task.SelectedApplicantID == "" &&
		(task.AppliedByMe || (task.ApplicationStatus == "open" && now.Before(task.ApplyDeadline)))
	task.CanViewResults = task.CanManage || task.SelectedByMe
	task.CanSubmitResult = task.SelectedByMe
	return &task, postOwner, nil
}

func (s *Server) applyTask(postID int64, applicantID string, appliedAt time.Time) error {
	task, ownerID, err := s.getTaskPostByID(postID, applicantID)
	if err != nil {
		return err
	}
	if task == nil {
		return errTaskNotFound
	}
	if ownerID == applicantID {
		return errTaskSelfApply
	}
	if task.ApplicationStatus != "open" || task.SelectedApplicantID != "" {
		return errTaskClosed
	}
	if !appliedAt.Before(task.ApplyDeadline) {
		return errTaskApplyEnded
	}
	_, err = s.db.Exec(
		`INSERT INTO task_applications (post_id, user_id, applied_at, withdrawn_at)
		 VALUES ($1, $2, $3, NULL)
		 ON CONFLICT (post_id, user_id) WHERE withdrawn_at IS NULL DO NOTHING`,
		postID,
		applicantID,
		appliedAt,
	)
	return err
}

func (s *Server) withdrawTaskApplication(postID int64, applicantID string, withdrawnAt time.Time) (bool, error) {
	result, err := s.db.Exec(
		`UPDATE task_applications
		    SET withdrawn_at = $3
		  WHERE post_id = $1 AND user_id = $2 AND withdrawn_at IS NULL`,
		postID,
		applicantID,
		withdrawnAt,
	)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (s *Server) listTaskApplications(postID int64) ([]TaskApplication, error) {
	rows, err := s.db.Query(
		`SELECT ta.id, ta.post_id, ta.user_id, u.username, u.icon_url, ta.applied_at
		   FROM task_applications ta
		   JOIN users u ON u.id = ta.user_id
		  WHERE ta.post_id = $1 AND ta.withdrawn_at IS NULL
		  ORDER BY ta.applied_at ASC`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applications := make([]TaskApplication, 0)
	for rows.Next() {
		var item TaskApplication
		if err := rows.Scan(&item.ID, &item.PostID, &item.UserID, &item.Username, &item.UserIcon, &item.AppliedAt); err != nil {
			return nil, err
		}
		applications = append(applications, item)
	}
	return applications, rows.Err()
}

func (s *Server) closeTaskApplications(postID int64, ownerID string) (bool, error) {
	result, err := s.db.Exec(
		`UPDATE task_posts tp
		    SET application_status = 'closed'
		   FROM posts p
		  WHERE tp.post_id = p.id AND tp.post_id = $1 AND p.user_id = $2`,
		postID,
		ownerID,
	)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func defaultTaskInvitation(postContent, workingHours string, startAt, endAt time.Time) string {
	return "你好，你已被选为该零工任务的候选人。\n\n任务内容：" + postContent +
		"\n时间范围：" + startAt.Format("2006-01-02 15:04") + " - " + endAt.Format("2006-01-02 15:04") +
		"\nWorking hours：" + workingHours +
		"\n如果你确认参与，请直接回复。"
}

func (s *Server) selectTaskApplicant(postID int64, ownerID, applicantID, invitationTemplate string, selectedAt time.Time) (int64, int64, string, error) {
	var (
		postContent   string
		taskStartAt   time.Time
		taskEndAt     time.Time
		workingHours  string
		ownerMatch    string
		applicationOK bool
	)
	err := s.db.QueryRow(
		`SELECT p.user_id, p.content, tp.start_at, tp.end_at, tp.working_hours,
		        EXISTS(
		          SELECT 1 FROM task_applications ta
		           WHERE ta.post_id = tp.post_id AND ta.user_id = $2 AND ta.withdrawn_at IS NULL
		        ) AS application_ok
		   FROM posts p
		   JOIN task_posts tp ON tp.post_id = p.id
		  WHERE p.id = $1`,
		postID,
		applicantID,
	).Scan(&ownerMatch, &postContent, &taskStartAt, &taskEndAt, &workingHours, &applicationOK)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, "", errTaskNotFound
		}
		return 0, 0, "", err
	}
	if ownerMatch != ownerID {
		return 0, 0, "", errTaskNotFound
	}
	if !applicationOK {
		return 0, 0, "", errTaskClosed
	}
	if strings.TrimSpace(invitationTemplate) == "" {
		invitationTemplate = defaultTaskInvitation(postContent, workingHours, taskStartAt, taskEndAt)
	}

	thread, err := s.ensureChatThread(ownerID, applicantID, selectedAt)
	if err != nil {
		return 0, 0, "", err
	}

	messageID, err := s.createChatMessage(thread.ID, nil, ownerID, invitationTemplate, selectedAt)
	if err != nil {
		return 0, 0, "", err
	}

	if _, err = s.db.Exec(
		`UPDATE task_posts
		    SET selected_applicant_id = $2,
		        selected_at = $3,
		        application_status = 'closed',
		        invitation_template = $4,
		        invitation_sent_at = $3
		  WHERE post_id = $1`,
		postID,
		applicantID,
		selectedAt,
		invitationTemplate,
	); err != nil {
		return 0, 0, "", err
	}
	return thread.ID, messageID, invitationTemplate, nil
}

func (s *Server) canAccessTaskResults(postID int64, userID string) (bool, bool, error) {
	task, _, err := s.getTaskPostByID(postID, userID)
	if err != nil {
		return false, false, err
	}
	if task == nil {
		return false, false, errTaskNotFound
	}
	return task.CanViewResults, task.CanSubmitResult, nil
}

func (s *Server) createTaskResult(postID int64, userID, note string, createdAt time.Time) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO task_results (post_id, user_id, note, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		postID,
		userID,
		note,
		createdAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Server) addTaskResultImage(resultID int64, fileURL string, createdAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO task_result_images (result_id, file_url, created_at)
		 VALUES ($1, $2, $3)`,
		resultID,
		fileURL,
		createdAt,
	)
	return err
}

func (s *Server) addTaskResultVideo(resultID int64, fileURL, posterURL string, createdAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO task_result_videos (result_id, file_url, poster_url, created_at)
		 VALUES ($1, $2, $3, $4)`,
		resultID,
		fileURL,
		posterURL,
		createdAt,
	)
	return err
}

func (s *Server) deleteTaskResult(resultID int64) error {
	_, err := s.db.Exec(`DELETE FROM task_results WHERE id = $1`, resultID)
	return err
}

func (s *Server) listTaskResults(postID int64, userID string) ([]TaskResult, error) {
	canView, _, err := s.canAccessTaskResults(postID, userID)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, errTaskForbidden
	}

	rows, err := s.db.Query(
		`SELECT tr.id, tr.post_id, tr.user_id, u.username, u.icon_url, tr.note, tr.created_at
		   FROM task_results tr
		   JOIN users u ON u.id = tr.user_id
		  WHERE tr.post_id = $1
		  ORDER BY tr.created_at DESC, tr.id DESC`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]TaskResult, 0)
	resultIDs := make([]int64, 0)
	for rows.Next() {
		var item TaskResult
		if err := rows.Scan(&item.ID, &item.PostID, &item.UserID, &item.Username, &item.UserIcon, &item.Note, &item.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, item)
		resultIDs = append(resultIDs, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(resultIDs) == 0 {
		return results, nil
	}

	imageRows, err := s.db.Query(
		`SELECT result_id, file_url FROM task_result_images
		  WHERE result_id = ANY($1)
		  ORDER BY id ASC`,
		pq.Array(resultIDs),
	)
	if err != nil {
		return nil, err
	}
	defer imageRows.Close()

	imageMap := make(map[int64][]string, len(resultIDs))
	for imageRows.Next() {
		var resultID int64
		var fileURL string
		if err := imageRows.Scan(&resultID, &fileURL); err != nil {
			return nil, err
		}
		imageMap[resultID] = append(imageMap[resultID], fileURL)
	}
	if err := imageRows.Err(); err != nil {
		return nil, err
	}

	videoRows, err := s.db.Query(
		`SELECT result_id, file_url, poster_url FROM task_result_videos
		  WHERE result_id = ANY($1)
		  ORDER BY id ASC`,
		pq.Array(resultIDs),
	)
	if err != nil {
		return nil, err
	}
	defer videoRows.Close()

	videoMap := make(map[int64][]string, len(resultIDs))
	videoItemMap := make(map[int64][]PostVideo, len(resultIDs))
	for videoRows.Next() {
		var resultID int64
		var fileURL string
		var posterURL string
		if err := videoRows.Scan(&resultID, &fileURL, &posterURL); err != nil {
			return nil, err
		}
		videoMap[resultID] = append(videoMap[resultID], fileURL)
		videoItemMap[resultID] = append(videoItemMap[resultID], PostVideo{URL: fileURL, PosterURL: posterURL})
	}
	if err := videoRows.Err(); err != nil {
		return nil, err
	}

	for i := range results {
		results[i].Images = imageMap[results[i].ID]
		results[i].Videos = videoMap[results[i].ID]
		results[i].VideoItems = videoItemMap[results[i].ID]
	}
	return results, nil
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
	var lastMessageID sql.NullInt64
	var lastMessageAt sql.NullTime
	err := s.db.QueryRow(
		`INSERT INTO chat_threads (user_low, user_high, created_at, last_message)
		 VALUES ($1, $2, $3, '')
		 ON CONFLICT (user_low, user_high)
		 DO UPDATE SET user_low = EXCLUDED.user_low
		 RETURNING id, user_low, user_high, created_at, last_message, last_message_id, last_message_at`,
		userLow,
		userHigh,
		createdAt,
	).Scan(
		&thread.ID,
		&thread.UserLow,
		&thread.UserHigh,
		&thread.CreatedAt,
		&thread.LastMessage,
		&lastMessageID,
		&lastMessageAt,
	)
	if err != nil {
		return nil, err
	}
	if lastMessageID.Valid {
		thread.LastMessageID = &lastMessageID.Int64
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
		        u.icon_url,
		        u.is_online,
		        u.last_active_device_type,
		        u.last_seen_at,
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
		var otherUserLastSeenAt sql.NullTime
		var lastMessageAt sql.NullTime
		if err := rows.Scan(
			&summary.ID,
			&summary.OtherUserID,
			&summary.OtherUsername,
			&summary.OtherUserIcon,
			&summary.OtherUserOnline,
			&summary.OtherUserDeviceType,
			&otherUserLastSeenAt,
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
		if otherUserLastSeenAt.Valid {
			summary.OtherUserLastSeenAt = &otherUserLastSeenAt.Time
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
	for i := range threads {
		isImplicitFriend, replyRequired, replyRequiredMessage, err := s.getChatReplyState(threads[i].ID, userID)
		if err != nil {
			return nil, false, err
		}
		threads[i].IsImplicitFriend = isImplicitFriend
		threads[i].ReplyRequired = replyRequired
		threads[i].ReplyRequiredMessage = replyRequiredMessage
	}
	return threads, hasMore, nil
}

func (s *Server) getChatSummary(userID string, threadID int64) (*ChatSummary, error) {
	var summary ChatSummary
	var lastMessageAt sql.NullTime
	var otherUserLastSeenAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT t.id,
		        CASE WHEN t.user_low = $1 THEN t.user_high ELSE t.user_low END AS other_id,
		        u.username,
		        u.icon_url,
		        u.is_online,
		        u.last_active_device_type,
		        u.last_seen_at,
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
		&summary.OtherUserIcon,
		&summary.OtherUserOnline,
		&summary.OtherUserDeviceType,
		&otherUserLastSeenAt,
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
	if otherUserLastSeenAt.Valid {
		summary.OtherUserLastSeenAt = &otherUserLastSeenAt.Time
	}
	isImplicitFriend, replyRequired, replyRequiredMessage, err := s.getChatReplyState(threadID, userID)
	if err != nil {
		return nil, err
	}
	summary.IsImplicitFriend = isImplicitFriend
	summary.ReplyRequired = replyRequired
	summary.ReplyRequiredMessage = replyRequiredMessage
	return &summary, nil
}

func (s *Server) listChatMessages(threadID int64, llmThreadID *int64, limit, offset int) ([]ChatMessage, bool, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	query := `SELECT m.id, m.thread_id, m.llm_thread_id, m.sender_id, u.username, u.icon_url, m.message_type, m.failed, m.content, m.markdown_entry_id, m.markdown_title, m.attachment, m.created_at, m.deleted_at, m.deleted_by
		   FROM chat_messages m
		   JOIN users u ON u.id = m.sender_id
		  WHERE m.thread_id = $1`
	args := []any{threadID}
	if llmThreadID != nil {
		query += ` AND m.llm_thread_id = $2 ORDER BY m.created_at ASC LIMIT $3 OFFSET $4`
		args = append(args, *llmThreadID, limit+1, offset)
	} else {
		query += ` ORDER BY m.created_at ASC LIMIT $2 OFFSET $3`
		args = append(args, limit+1, offset)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	messages := make([]ChatMessage, 0, limit+1)
	for rows.Next() {
		var msg ChatMessage
		var llmThreadIDValue sql.NullInt64
		var markdownEntryID sql.NullInt64
		var attachmentJSON sql.NullString
		var deletedAt sql.NullTime
		var deletedBy sql.NullString
		if err := rows.Scan(
			&msg.ID,
			&msg.ThreadID,
			&llmThreadIDValue,
			&msg.SenderID,
			&msg.SenderUsername,
			&msg.SenderIcon,
			&msg.MessageType,
			&msg.Failed,
			&msg.Content,
			&markdownEntryID,
			&msg.MarkdownTitle,
			&attachmentJSON,
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
		if llmThreadIDValue.Valid {
			msg.LLMThreadID = &llmThreadIDValue.Int64
		}
		if markdownEntryID.Valid {
			msg.MarkdownEntryID = &markdownEntryID.Int64
		}
		if deletedBy.Valid {
			msg.DeletedBy = deletedBy.String
		}
		if attachmentJSON.Valid && attachmentJSON.String != "" {
			var att ChatMessageAttachment
			if jsonErr := json.Unmarshal([]byte(attachmentJSON.String), &att); jsonErr == nil {
				msg.Attachment = &att
			}
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

func (s *Server) markChatRead(threadID int64, userID string, readAt time.Time, lastReadMessageID *int64) error {
	_, err := s.db.Exec(
		`INSERT INTO chat_reads (thread_id, user_id, last_read_at, last_read_message_id)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (thread_id, user_id)
		 DO UPDATE SET last_read_at = EXCLUDED.last_read_at,
		               last_read_message_id = CASE
		                   WHEN chat_reads.last_read_message_id IS NULL THEN EXCLUDED.last_read_message_id
		                   WHEN EXCLUDED.last_read_message_id IS NULL THEN chat_reads.last_read_message_id
		                   WHEN EXCLUDED.last_read_message_id > chat_reads.last_read_message_id THEN EXCLUDED.last_read_message_id
		                   ELSE chat_reads.last_read_message_id
		               END`,
		threadID,
		userID,
		readAt,
		lastReadMessageID,
	)
	return err
}

func (s *Server) upsertChatMemberStateViewed(threadID int64, userID string, openedAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO chat_member_state (thread_id, user_id, last_opened_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $3, $3)
		 ON CONFLICT (thread_id, user_id)
		 DO UPDATE SET last_opened_at = EXCLUDED.last_opened_at,
		               updated_at = EXCLUDED.updated_at`,
		threadID,
		userID,
		openedAt,
	)
	return err
}

func (s *Server) getChatParticipants(threadID int64) (string, string, error) {
	var userLow, userHigh string
	err := s.db.QueryRow(
		`SELECT user_low, user_high FROM chat_threads WHERE id = $1`,
		threadID,
	).Scan(&userLow, &userHigh)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil
		}
		return "", "", err
	}
	return userLow, userHigh, nil
}

func (s *Server) getChatCounterparty(threadID int64, userID string) (string, error) {
	userLow, userHigh, err := s.getChatParticipants(threadID)
	if err != nil {
		return "", err
	}
	switch userID {
	case userLow:
		return userHigh, nil
	case userHigh:
		return userLow, nil
	default:
		return "", nil
	}
}

func (s *Server) getChatImplicitFriendState(threadID int64) (bool, error) {
	var distinctSenders int
	err := s.db.QueryRow(
		`SELECT COUNT(DISTINCT sender_id)
		   FROM chat_messages
		  WHERE thread_id = $1
		    AND deleted_at IS NULL`,
		threadID,
	).Scan(&distinctSenders)
	if err != nil {
		return false, err
	}
	return distinctSenders >= 2, nil
}

func (s *Server) getLastUndeletedChatMessageSender(threadID int64) (string, error) {
	var senderID string
	err := s.db.QueryRow(
		`SELECT sender_id
		   FROM chat_messages
		  WHERE thread_id = $1
		    AND deleted_at IS NULL
		  ORDER BY created_at DESC, id DESC
		  LIMIT 1`,
		threadID,
	).Scan(&senderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return senderID, nil
}

func (s *Server) getChatReplyState(threadID int64, userID string) (bool, bool, string, error) {
	responderUserID, err := s.getAIResponderForChat(threadID, userID)
	if err != nil {
		return false, false, "", err
	}
	if responderUserID != "" {
		return true, false, "", nil
	}

	isImplicitFriend, err := s.getChatImplicitFriendState(threadID)
	if err != nil {
		return false, false, "", err
	}
	if isImplicitFriend {
		return true, false, "", nil
	}

	lastSenderID, err := s.getLastUndeletedChatMessageSender(threadID)
	if err != nil {
		return false, false, "", err
	}
	if lastSenderID == userID {
		return false, true, "你已发送首条消息，请等待对方回复后再继续发送", nil
	}
	return false, false, "", nil
}

func (s *Server) listLLMThreads(chatThreadID int64, ownerUserID string) ([]LLMThread, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.chat_thread_id, t.owner_user_id, t.bot_user_id, COALESCE(t.llm_config_id, b.llm_config_id), COALESCE(c.name, ''), COALESCE(c.model, ''), t.title, t.created_at, t.updated_at, t.last_message_at
		   FROM llm_threads t
		   LEFT JOIN bot_users b ON b.bot_user_id = t.bot_user_id
		   LEFT JOIN llm_configs c ON c.id = COALESCE(t.llm_config_id, b.llm_config_id)
		  WHERE t.chat_thread_id = $1 AND t.owner_user_id = $2
		  ORDER BY updated_at DESC, id DESC`,
		chatThreadID,
		ownerUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]LLMThread, 0)
	for rows.Next() {
		var item LLMThread
		var llmConfigID sql.NullInt64
		var lastMessageAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.ChatThreadID, &item.OwnerUserID, &item.BotUserID, &llmConfigID, &item.ConfigName, &item.ConfigModel, &item.Title, &item.CreatedAt, &item.UpdatedAt, &lastMessageAt); err != nil {
			return nil, err
		}
		if llmConfigID.Valid {
			item.LLMConfigID = &llmConfigID.Int64
		}
		if lastMessageAt.Valid {
			item.LastMessageAt = &lastMessageAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) getLLMThread(chatThreadID int64, ownerUserID string, llmThreadID int64) (*LLMThread, error) {
	var item LLMThread
	var llmConfigID sql.NullInt64
	var lastMessageAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT t.id, t.chat_thread_id, t.owner_user_id, t.bot_user_id, COALESCE(t.llm_config_id, b.llm_config_id), COALESCE(c.name, ''), COALESCE(c.model, ''), t.title, t.created_at, t.updated_at, t.last_message_at
		   FROM llm_threads t
		   LEFT JOIN bot_users b ON b.bot_user_id = t.bot_user_id
		   LEFT JOIN llm_configs c ON c.id = COALESCE(t.llm_config_id, b.llm_config_id)
		  WHERE t.id = $1 AND t.chat_thread_id = $2 AND t.owner_user_id = $3`,
		llmThreadID, chatThreadID, ownerUserID,
	).Scan(&item.ID, &item.ChatThreadID, &item.OwnerUserID, &item.BotUserID, &llmConfigID, &item.ConfigName, &item.ConfigModel, &item.Title, &item.CreatedAt, &item.UpdatedAt, &lastMessageAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if llmConfigID.Valid {
		item.LLMConfigID = &llmConfigID.Int64
	}
	if lastMessageAt.Valid {
		item.LastMessageAt = &lastMessageAt.Time
	}
	return &item, nil
}

func (s *Server) createLLMThread(chatThreadID int64, ownerUserID, botUserID, title string, now time.Time) (*LLMThread, error) {
	if strings.TrimSpace(title) == "" {
		title = "新话题"
	}
	botUser, err := s.getBotUserByUserID(botUserID)
	if err != nil {
		return nil, err
	}
	if botUser == nil {
		return nil, nil
	}
	var itemID int64
	err = s.db.QueryRow(
		`INSERT INTO llm_threads (chat_thread_id, owner_user_id, bot_user_id, llm_config_id, title, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)
		 RETURNING id`,
		chatThreadID, ownerUserID, botUserID, botUser.LLMConfigID, title, now,
	).Scan(&itemID)
	if err != nil {
		return nil, err
	}
	return s.getLLMThread(chatThreadID, ownerUserID, itemID)
}

func (s *Server) ensureDefaultLLMThread(chatThreadID int64, ownerUserID, botUserID string, now time.Time) (*LLMThread, error) {
	var item LLMThread
	var llmConfigID sql.NullInt64
	var lastMessageAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT t.id, t.chat_thread_id, t.owner_user_id, t.bot_user_id, COALESCE(t.llm_config_id, b.llm_config_id), COALESCE(c.name, ''), COALESCE(c.model, ''), t.title, t.created_at, t.updated_at, t.last_message_at
		   FROM llm_threads t
		   LEFT JOIN bot_users b ON b.bot_user_id = t.bot_user_id
		   LEFT JOIN llm_configs c ON c.id = COALESCE(t.llm_config_id, b.llm_config_id)
		  WHERE t.chat_thread_id = $1 AND t.owner_user_id = $2 AND t.bot_user_id = $3
		  ORDER BY updated_at DESC, id DESC
		  LIMIT 1`,
		chatThreadID, ownerUserID, botUserID,
	).Scan(&item.ID, &item.ChatThreadID, &item.OwnerUserID, &item.BotUserID, &llmConfigID, &item.ConfigName, &item.ConfigModel, &item.Title, &item.CreatedAt, &item.UpdatedAt, &lastMessageAt)
	if err == nil {
		if llmConfigID.Valid {
			item.LLMConfigID = &llmConfigID.Int64
		}
		if lastMessageAt.Valid {
			item.LastMessageAt = &lastMessageAt.Time
		}
		return &item, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return s.createLLMThread(chatThreadID, ownerUserID, botUserID, "新话题", now)
}

func (s *Server) updateLLMThreadTitle(chatThreadID int64, ownerUserID string, llmThreadID int64, title string, now time.Time) (*LLMThread, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "新话题"
	}
	var updatedID int64
	err := s.db.QueryRow(
		`UPDATE llm_threads
		    SET title = $4, updated_at = $5
		  WHERE id = $1 AND chat_thread_id = $2 AND owner_user_id = $3
		  RETURNING id`,
		llmThreadID, chatThreadID, ownerUserID, title, now,
	).Scan(&updatedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return s.getLLMThread(chatThreadID, ownerUserID, updatedID)
}

func (s *Server) deleteLLMThread(chatThreadID int64, ownerUserID string, llmThreadID int64) (bool, error) {
	result, err := s.db.Exec(
		`DELETE FROM llm_threads
		  WHERE id = $1 AND chat_thread_id = $2 AND owner_user_id = $3`,
		llmThreadID,
		chatThreadID,
		ownerUserID,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *Server) updateLLMThreadConfig(chatThreadID int64, ownerUserID string, llmThreadID, llmConfigID int64, now time.Time) (*LLMThread, error) {
	var configName, configModel string
	if err := s.db.QueryRow(
		`SELECT name, model FROM llm_configs WHERE id = $1 AND (owner_user_id = $2 OR shared = TRUE)`,
		llmConfigID,
		ownerUserID,
	).Scan(&configName, &configModel); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var updatedID int64
	err := s.db.QueryRow(
		`UPDATE llm_threads
		    SET llm_config_id = $4, updated_at = $5
		  WHERE id = $1 AND chat_thread_id = $2 AND owner_user_id = $3
		  RETURNING id`,
		llmThreadID,
		chatThreadID,
		ownerUserID,
		llmConfigID,
		now,
	).Scan(&updatedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item, err := s.getLLMThread(chatThreadID, ownerUserID, updatedID)
	if err != nil || item == nil {
		return item, err
	}
	if strings.TrimSpace(item.ConfigName) == "" {
		item.ConfigName = configName
	}
	if strings.TrimSpace(item.ConfigModel) == "" {
		item.ConfigModel = configModel
	}
	return item, nil
}

func buildLLMThreadTitle(content string, fallback string) string {
	replacer := strings.NewReplacer(
		"\r", " ",
		"\n", " ",
		"`", "",
		"#", "",
		"*", "",
		">", "",
		"[", "",
		"]", "",
		"(", "",
		")", "",
	)
	text := strings.TrimSpace(replacer.Replace(content))
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		if strings.TrimSpace(fallback) != "" {
			return strings.TrimSpace(fallback)
		}
		return "新话题"
	}
	runes := []rune(text)
	if len(runes) > 24 {
		return string(runes[:24]) + "..."
	}
	return text
}

func (s *Server) createChatMessage(threadID int64, llmThreadID *int64, senderID, content string, createdAt time.Time) (int64, error) {
	return s.createChatMessageWithOptions(threadID, llmThreadID, senderID, "text", false, content, nil, "", createdAt)
}

func (s *Server) createChatMessageWithMetadata(threadID int64, llmThreadID *int64, senderID, messageType, content string, markdownEntryID *int64, markdownTitle string, createdAt time.Time) (int64, error) {
	return s.createChatMessageWithOptions(threadID, llmThreadID, senderID, messageType, false, content, markdownEntryID, markdownTitle, createdAt)
}

func (s *Server) createAttachmentChatMessage(threadID int64, senderID, preview string, att ChatMessageAttachment, createdAt time.Time) (int64, error) {
	attJSON, err := json.Marshal(att)
	if err != nil {
		return 0, err
	}
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
		`INSERT INTO chat_messages (thread_id, sender_id, message_type, failed, content, attachment, created_at)
		 VALUES ($1, $2, 'attachment', FALSE, $3, $4, $5)
		 RETURNING id`,
		threadID,
		senderID,
		preview,
		string(attJSON),
		createdAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	if _, err = tx.Exec(
		`UPDATE chat_threads
		    SET last_message = $1, last_message_id = $2, last_message_at = $3
		  WHERE id = $4`,
		preview,
		id,
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

func (s *Server) createChatMessageWithOptions(threadID int64, llmThreadID *int64, senderID, messageType string, failed bool, content string, markdownEntryID *int64, markdownTitle string, createdAt time.Time) (int64, error) {
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
		`INSERT INTO chat_messages (thread_id, llm_thread_id, sender_id, message_type, failed, content, markdown_entry_id, markdown_title, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id`,
		threadID,
		llmThreadID,
		senderID,
		messageType,
		failed,
		content,
		markdownEntryID,
		markdownTitle,
		createdAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	if _, err = tx.Exec(
		`UPDATE chat_threads
		    SET last_message = $1, last_message_id = $2, last_message_at = $3
		  WHERE id = $4`,
		content,
		id,
		createdAt,
		threadID,
	); err != nil {
		return 0, err
	}
	if llmThreadID != nil {
		var currentTitle string
		if err = tx.QueryRow(`SELECT title FROM llm_threads WHERE id = $1`, *llmThreadID).Scan(&currentTitle); err != nil {
			return 0, err
		}
		if strings.TrimSpace(currentTitle) == "" || strings.TrimSpace(currentTitle) == "新话题" {
			nextTitle := buildLLMThreadTitle(content, markdownTitle)
			if _, err = tx.Exec(
				`UPDATE llm_threads
				    SET title = $1, last_message_at = $2, updated_at = $2
				  WHERE id = $3`,
				nextTitle,
				createdAt,
				*llmThreadID,
			); err != nil {
				return 0, err
			}
		} else {
			if _, err = tx.Exec(
				`UPDATE llm_threads
			    SET last_message_at = $1, updated_at = $1
			  WHERE id = $2`,
				createdAt,
				*llmThreadID,
			); err != nil {
				return 0, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Server) getChatMessageByID(messageID int64) (*ChatMessage, error) {
	var msg ChatMessage
	var llmThreadID sql.NullInt64
	var markdownEntryID sql.NullInt64
	var deletedAt sql.NullTime
	var deletedBy sql.NullString
	var attachmentJSON sql.NullString
	err := s.db.QueryRow(
		`SELECT m.id, m.thread_id, m.llm_thread_id, m.sender_id, u.username, u.icon_url, m.message_type, m.failed, m.content, m.markdown_entry_id, m.markdown_title, m.attachment, m.created_at, m.deleted_at, m.deleted_by
		   FROM chat_messages m
		   JOIN users u ON u.id = m.sender_id
		  WHERE m.id = $1`,
		messageID,
	).Scan(
		&msg.ID,
		&msg.ThreadID,
		&llmThreadID,
		&msg.SenderID,
		&msg.SenderUsername,
		&msg.SenderIcon,
		&msg.MessageType,
		&msg.Failed,
		&msg.Content,
		&markdownEntryID,
		&msg.MarkdownTitle,
		&attachmentJSON,
		&msg.CreatedAt,
		&deletedAt,
		&deletedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if deletedAt.Valid {
		msg.DeletedAt = &deletedAt.Time
		msg.Deleted = true
		msg.Content = ""
	}
	if llmThreadID.Valid {
		msg.LLMThreadID = &llmThreadID.Int64
	}
	if markdownEntryID.Valid {
		msg.MarkdownEntryID = &markdownEntryID.Int64
	}
	if deletedBy.Valid {
		msg.DeletedBy = deletedBy.String
	}
	if attachmentJSON.Valid && attachmentJSON.String != "" {
		var att ChatMessageAttachment
		if jsonErr := json.Unmarshal([]byte(attachmentJSON.String), &att); jsonErr == nil {
			msg.Attachment = &att
		}
	}
	return &msg, nil
}

func (s *Server) listRecentChatMessages(threadID int64, llmThreadID *int64, limit int) ([]ChatMessage, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `SELECT m.id, m.thread_id, m.llm_thread_id, m.sender_id, u.username, u.icon_url, m.message_type, m.failed, m.content, m.markdown_entry_id, m.markdown_title, m.attachment, m.created_at, m.deleted_at, m.deleted_by
		   FROM chat_messages m
		   JOIN users u ON u.id = m.sender_id
		  WHERE m.thread_id = $1`
	args := []any{threadID}
	if llmThreadID != nil {
		query += ` AND m.llm_thread_id = $2 ORDER BY m.created_at DESC LIMIT $3`
		args = append(args, *llmThreadID, limit)
	} else {
		query += ` ORDER BY m.created_at DESC LIMIT $2`
		args = append(args, limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ChatMessage, 0, limit)
	for rows.Next() {
		var msg ChatMessage
		var llmThreadIDValue sql.NullInt64
		var markdownEntryID sql.NullInt64
		var attachmentJSON sql.NullString
		var deletedAt sql.NullTime
		var deletedBy sql.NullString
		if err := rows.Scan(
			&msg.ID,
			&msg.ThreadID,
			&llmThreadIDValue,
			&msg.SenderID,
			&msg.SenderUsername,
			&msg.SenderIcon,
			&msg.MessageType,
			&msg.Failed,
			&msg.Content,
			&markdownEntryID,
			&msg.MarkdownTitle,
			&attachmentJSON,
			&msg.CreatedAt,
			&deletedAt,
			&deletedBy,
		); err != nil {
			return nil, err
		}
		if deletedAt.Valid {
			msg.DeletedAt = &deletedAt.Time
			msg.Deleted = true
		}
		if llmThreadIDValue.Valid {
			msg.LLMThreadID = &llmThreadIDValue.Int64
		}
		if markdownEntryID.Valid {
			msg.MarkdownEntryID = &markdownEntryID.Int64
		}
		if deletedBy.Valid {
			msg.DeletedBy = deletedBy.String
		}
		if attachmentJSON.Valid && attachmentJSON.String != "" {
			var att ChatMessageAttachment
			if jsonErr := json.Unmarshal([]byte(attachmentJSON.String), &att); jsonErr == nil {
				msg.Attachment = &att
			}
		}
		items = append(items, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items, nil
}

func (s *Server) findRetrySourceMessage(threadID int64, targetMessage *ChatMessage) (*ChatMessage, error) {
	if targetMessage == nil {
		return nil, nil
	}

	query := `SELECT m.id, m.thread_id, m.llm_thread_id, m.sender_id, u.username, u.icon_url, m.message_type, m.failed, m.content, m.markdown_entry_id, m.markdown_title, m.attachment, m.created_at, m.deleted_at, m.deleted_by
	   FROM chat_messages m
	   JOIN users u ON u.id = m.sender_id
	  WHERE m.thread_id = $1
	    AND m.sender_id <> $2
	    AND m.deleted_at IS NULL
	    AND (m.created_at < $3 OR (m.created_at = $3 AND m.id < $4))`
	args := []any{threadID, targetMessage.SenderID, targetMessage.CreatedAt, targetMessage.ID}
	if targetMessage.LLMThreadID != nil {
		query += ` AND m.llm_thread_id = $5`
		args = append(args, *targetMessage.LLMThreadID)
	} else {
		query += ` AND m.llm_thread_id IS NULL`
	}
	query += ` ORDER BY m.created_at DESC, m.id DESC LIMIT 1`

	row := s.db.QueryRow(query, args...)
	var msg ChatMessage
	var llmThreadID sql.NullInt64
	var markdownEntryID sql.NullInt64
	var attachmentJSON sql.NullString
	var deletedAt sql.NullTime
	var deletedBy sql.NullString
	err := row.Scan(
		&msg.ID,
		&msg.ThreadID,
		&llmThreadID,
		&msg.SenderID,
		&msg.SenderUsername,
		&msg.SenderIcon,
		&msg.MessageType,
		&msg.Failed,
		&msg.Content,
		&markdownEntryID,
		&msg.MarkdownTitle,
		&attachmentJSON,
		&msg.CreatedAt,
		&deletedAt,
		&deletedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if llmThreadID.Valid {
		msg.LLMThreadID = &llmThreadID.Int64
	}
	if markdownEntryID.Valid {
		msg.MarkdownEntryID = &markdownEntryID.Int64
	}
	if deletedAt.Valid {
		msg.DeletedAt = &deletedAt.Time
		msg.Deleted = true
	}
	if deletedBy.Valid {
		msg.DeletedBy = deletedBy.String
	}
	if attachmentJSON.Valid && attachmentJSON.String != "" {
		var att ChatMessageAttachment
		if jsonErr := json.Unmarshal([]byte(attachmentJSON.String), &att); jsonErr == nil {
			msg.Attachment = &att
		}
	}
	return &msg, nil
}

func (s *Server) markChatMessageFailedResolved(threadID, messageID int64, deletedAt time.Time) (bool, error) {
	result, err := s.db.Exec(
		`UPDATE chat_messages
		    SET deleted_at = $1, deleted_by = 'retry'
		  WHERE id = $2 AND thread_id = $3 AND failed = TRUE AND deleted_at IS NULL`,
		deletedAt,
		messageID,
		threadID,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
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

func (s *Server) createPushDelivery(messageID int64, userID, deviceID, pushToken, status, errorMessage string, now time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO push_deliveries (message_id, user_id, device_id, push_token, provider, status, apns_id, error_message, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'apns', $5, '', $6, $7, $7)`,
		messageID,
		userID,
		strings.TrimSpace(deviceID),
		sanitizePushToken(pushToken),
		strings.TrimSpace(status),
		strings.TrimSpace(errorMessage),
		now,
	)
	return err
}

func (s *Server) claimPendingPushDeliveries(limit int, now time.Time) ([]PushDelivery, error) {
	if limit <= 0 {
		limit = 20
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	rows, err := tx.Query(
		`SELECT id, message_id, user_id, device_id, push_token, provider, status, apns_id, error_message, created_at, updated_at
		   FROM push_deliveries
		  WHERE status = 'pending'
		  ORDER BY created_at ASC, id ASC
		  LIMIT $1
		  FOR UPDATE SKIP LOCKED`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PushDelivery, 0, limit)
	ids := make([]int64, 0, limit)
	for rows.Next() {
		var item PushDelivery
		if err := rows.Scan(
			&item.ID,
			&item.MessageID,
			&item.UserID,
			&item.DeviceID,
			&item.PushToken,
			&item.Provider,
			&item.Status,
			&item.APNSID,
			&item.ErrorMessage,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
		ids = append(ids, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		tx = nil
		return items, nil
	}

	if _, err := tx.Exec(
		`UPDATE push_deliveries
		    SET status = 'processing',
		        updated_at = $2
		  WHERE id = ANY($1)`,
		pq.Array(ids),
		now,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	for i := range items {
		items[i].Status = "processing"
		items[i].UpdatedAt = now
	}
	return items, nil
}

func (s *Server) updatePushDeliveryResult(id int64, status, apnsID, errorMessage string, now time.Time) error {
	_, err := s.db.Exec(
		`UPDATE push_deliveries
		    SET status = $2,
		        apns_id = $3,
		        error_message = $4,
		        updated_at = $5
		  WHERE id = $1`,
		id,
		strings.TrimSpace(status),
		strings.TrimSpace(apnsID),
		strings.TrimSpace(errorMessage),
		now,
	)
	return err
}
