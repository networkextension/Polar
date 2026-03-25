package dock

import (
	"context"
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	geoip2 "github.com/oschwald/geoip2-golang/v2"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	db              *sql.DB
	redis           *redis.Client
	router          *gin.Engine
	addr            string
	redisPrefix     string
	markdownDir     string
	uploadDir       string
	geoLiteDBPath   string
	geoIPReader     *geoip2.Reader
	webAuthn        *webauthn.WebAuthn
	passkeyAuto     bool
	passkeyRPID     string
	passkeyOrigin   string
	passkeyRPName   string
	passkeySessions map[string]passkeySession
	passkeyMu       sync.Mutex
	wsHub           *wsHub
	workDir         string
	aiAgent         *aiAgent
}

func NewServer(cfg Config) (*Server, error) {
	db, err := openDB(cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	server := &Server{
		db:            db,
		addr:          cfg.Addr,
		redisPrefix:   cfg.RedisPrefix,
		markdownDir:   cfg.MarkdownDir,
		uploadDir:     cfg.UploadDir,
		geoLiteDBPath: cfg.GeoLiteDBPath,
		redis: redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		}),
	}

	workDir, err := os.Getwd()
	if err == nil {
		server.workDir = workDir
	}

	if err := server.redis.Ping(context.Background()).Err(); err != nil {
		_ = db.Close()
		return nil, err
	}

	if cfg.GeoLiteDBPath != "" {
		reader, err := geoip2.Open(cfg.GeoLiteDBPath)
		if err != nil {
			log.Printf("geolite disabled: %v", err)
		} else {
			server.geoIPReader = reader
		}
	}

	server.passkeyRPID = cfg.PasskeyRPID
	server.passkeyOrigin = cfg.PasskeyOrigin
	server.passkeyRPName = cfg.PasskeyRPName
	server.passkeyAuto = cfg.PasskeyOrigin == DefaultPasskeyOrigin && cfg.PasskeyRPID == DefaultPasskeyRPID

	rpID := cfg.PasskeyRPID
	if rpID == DefaultPasskeyRPID {
		if parsed, err := url.Parse(cfg.PasskeyOrigin); err == nil && parsed.Host != "" {
			host := parsed.Host
			if strings.Contains(host, ":") {
				if splitHost, _, err := net.SplitHostPort(host); err == nil && splitHost != "" {
					host = splitHost
				}
			}
			if host != "" {
				rpID = host
			}
		}
	}

	webAuthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: cfg.PasskeyRPName,
		RPID:          rpID,
		RPOrigins:     []string{cfg.PasskeyOrigin},
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	server.webAuthn = webAuthn
	server.passkeySessions = make(map[string]passkeySession)
	server.wsHub = newWSHub()
	server.wsHub.onPresenceChanged = server.handlePresenceChange
	go server.wsHub.run()

	if err := server.ensureSystemUser(); err != nil {
		_ = db.Close()
		return nil, err
	}

	server.aiAgent = newAIAgent(server, cfg)
	if server.aiAgent != nil {
		go server.aiAgent.run()
	}

	server.router = gin.Default()
	// Increase max upload size for video files (default gin limit is 32 MiB)
	server.router.MaxMultipartMemory = 512 << 20 // 512 MiB
	server.router.Use(corsMiddleware())
	if server.uploadDir != "" {
		server.router.Static("/uploads", server.uploadDir)
	}
	server.registerRoutes()

	return server, nil
}

func (s *Server) Run() error {
	return s.router.Run(s.addr)
}

func (s *Server) Close() error {
	if s.aiAgent != nil {
		s.aiAgent.stop()
	}
	if s.geoIPReader != nil {
		_ = s.geoIPReader.Close()
	}
	if s.redis != nil {
		_ = s.redis.Close()
	}
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Server) ensureSystemUser() error {
	if s == nil {
		return errors.New("server is nil")
	}
	user, err := s.getUserByID(systemUserID)
	if err != nil {
		return err
	}
	if user != nil {
		return nil
	}

	password, err := hashPassword(generateSessionID())
	if err != nil {
		return err
	}

	now := time.Now()
	return s.createUser(&User{
		ID:        systemUserID,
		Username:  systemUsername,
		Email:     systemUserEmail,
		Password:  password,
		Role:      "admin",
		Bio:       "站内 AI 助理",
		CreatedAt: now,
	})
}

func (s *Server) registerRoutes() {
	tmpl := template.Must(template.New("layout").Parse(layoutTemplate))
	template.Must(tmpl.New("login").Parse(loginTemplate))
	template.Must(tmpl.New("register").Parse(registerTemplate))
	template.Must(tmpl.New("dashboard").Parse(dashboardTemplate))
	s.router.SetHTMLTemplate(tmpl)

	s.router.GET("/", func(c *gin.Context) {
		sessionID, _ := c.Cookie(SessionCookieName)
		if sessionID != "" && s.getSession(sessionID) != nil {
			c.Redirect(http.StatusFound, "/dashboard")
			return
		}
		c.Redirect(http.StatusFound, "/login")
	})

	s.router.GET("/login", s.GuestMiddleware(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "login", gin.H{"Title": "登录"})
	})

	s.router.GET("/register", s.GuestMiddleware(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "register", gin.H{"Title": "注册"})
	})

	s.router.GET("/dashboard", s.AuthMiddleware(), func(c *gin.Context) {
		username, _ := c.Get("username")
		userID, _ := c.Get("user_id")
		role, _ := c.Get("role")
		roleLabel := "普通用户组"
		if roleStr, ok := role.(string); ok && roleStr == "admin" {
			roleLabel = "管理用户组"
		}
		c.HTML(http.StatusOK, "dashboard", gin.H{
			"Title":     "控制台",
			"Username":  username,
			"UserID":    userID,
			"Role":      role,
			"RoleLabel": roleLabel,
			"LoginTime": time.Now().Format("2006-01-02 15:04:05"),
		})
	})
	s.router.GET("/ws/chat", s.handleChatWS)

	api := s.router.Group("/api")
	{
		api.POST("/register", s.handleRegister)
		api.POST("/login", s.handleLogin)
		api.POST("/logout", s.handleLogout)
		api.POST("/passkey/register/begin", s.AuthMiddleware(), s.handlePasskeyRegisterBegin)
		api.POST("/passkey/register/finish", s.AuthMiddleware(), s.handlePasskeyRegisterFinish)
		api.POST("/passkey/login/begin", s.GuestMiddleware(), s.handlePasskeyLoginBegin)
		api.POST("/passkey/login/finish", s.GuestMiddleware(), s.handlePasskeyLoginFinish)
		api.POST("/user/icon", s.AuthMiddleware(), s.handleUserIconUpload)
		api.PUT("/users/me/profile", s.AuthMiddleware(), s.handleMyProfileUpdate)
		api.GET("/users/:id/profile", s.AuthMiddleware(), s.handleUserProfileGet)
		api.POST("/users/:id/recommendations", s.AuthMiddleware(), s.handleProfileRecommendationUpsert)
		api.GET("/site-settings", s.handleSiteSettingsGet)
		api.PUT("/site-settings", s.AuthMiddleware(), s.AdminMiddleware(), s.handleSiteSettingsUpdate)
		api.POST("/site-settings/icon", s.AuthMiddleware(), s.AdminMiddleware(), s.handleSiteIconUpload)
		api.POST("/site-settings/apple-push-cert", s.AuthMiddleware(), s.AdminMiddleware(), s.handleApplePushCertificateUpload)
		api.DELETE("/site-settings/apple-push-cert", s.AuthMiddleware(), s.AdminMiddleware(), s.handleApplePushCertificateDelete)
		api.GET("/llm-configs", s.AuthMiddleware(), s.handleLLMConfigList)
		api.GET("/llm-configs/available", s.AuthMiddleware(), s.handleAvailableLLMConfigList)
		api.POST("/llm-configs/test", s.AuthMiddleware(), s.handleLLMConfigTest)
		api.POST("/llm-configs", s.AuthMiddleware(), s.handleLLMConfigCreate)
		api.GET("/llm-configs/shared/:shareId", s.AuthMiddleware(), s.handleLLMConfigGetByShareID)
		api.PUT("/llm-configs/:id", s.AuthMiddleware(), s.handleLLMConfigUpdate)
		api.DELETE("/llm-configs/:id", s.AuthMiddleware(), s.handleLLMConfigDelete)
		api.GET("/bots", s.AuthMiddleware(), s.handleBotUserList)
		api.POST("/bots", s.AuthMiddleware(), s.handleBotUserCreate)
		api.PUT("/bots/:id", s.AuthMiddleware(), s.handleBotUserUpdate)
		api.DELETE("/bots/:id", s.AuthMiddleware(), s.handleBotUserDelete)
		api.GET("/me", s.AuthMiddleware(), s.handleMe)
		api.GET("/login-history", s.AuthMiddleware(), s.handleLoginHistory)
		api.POST("/markdown", s.AuthMiddleware(), s.handleMarkdownSubmit)
		api.GET("/markdown", s.AuthMiddleware(), s.handleMarkdownList)
		api.GET("/public/markdowns", s.handlePublicMarkdownList)
		api.GET("/public/markdown/:id", s.handlePublicMarkdownRead)
		api.GET("/markdown/:id", s.AuthMiddleware(), s.handleMarkdownRead)
		api.PUT("/markdown/:id", s.AuthMiddleware(), s.handleMarkdownUpdate)
		api.DELETE("/markdown/:id", s.AuthMiddleware(), s.handleMarkdownDelete)
		api.GET("/tags", s.AuthMiddleware(), s.handleTagList)
		api.POST("/tags", s.AuthMiddleware(), s.AdminMiddleware(), s.handleTagCreate)
		api.PUT("/tags/:id", s.AuthMiddleware(), s.AdminMiddleware(), s.handleTagUpdate)
		api.DELETE("/tags/:id", s.AuthMiddleware(), s.AdminMiddleware(), s.handleTagDelete)
		api.POST("/posts", s.AuthMiddleware(), s.handlePostCreate)
		api.GET("/posts", s.AuthMiddleware(), s.handlePostList)
		api.GET("/posts/:id", s.AuthMiddleware(), s.handlePostRead)
		api.DELETE("/posts/:id", s.AuthMiddleware(), s.handlePostDelete)
		api.POST("/posts/:id/like", s.AuthMiddleware(), s.handlePostLike)
		api.DELETE("/posts/:id/like", s.AuthMiddleware(), s.handlePostUnlike)
		api.POST("/posts/:id/replies", s.AuthMiddleware(), s.handleReplyCreate)
		api.GET("/posts/:id/replies", s.AuthMiddleware(), s.handleReplyList)
		api.POST("/tasks/:id/apply", s.AuthMiddleware(), s.handleTaskApply)
		api.DELETE("/tasks/:id/apply", s.AuthMiddleware(), s.handleTaskWithdraw)
		api.GET("/tasks/:id/applications", s.AuthMiddleware(), s.handleTaskApplications)
		api.POST("/tasks/:id/close", s.AuthMiddleware(), s.handleTaskClose)
		api.POST("/tasks/:id/select-candidate", s.AuthMiddleware(), s.handleTaskSelectCandidate)
		api.GET("/tasks/:id/results", s.AuthMiddleware(), s.handleTaskResultsList)
		api.POST("/tasks/:id/results", s.AuthMiddleware(), s.handleTaskResultCreate)
		api.GET("/chats", s.AuthMiddleware(), s.handleChatList)
		api.POST("/chats/start", s.AuthMiddleware(), s.handleChatStart)
		api.GET("/chats/:id/llm-threads", s.AuthMiddleware(), s.handleChatLLMThreads)
		api.POST("/chats/:id/llm-threads", s.AuthMiddleware(), s.handleChatLLMThreadCreate)
		api.PUT("/chats/:id/llm-threads/:threadId", s.AuthMiddleware(), s.handleChatLLMThreadUpdate)
		api.PUT("/chats/:id/llm-threads/:threadId/config", s.AuthMiddleware(), s.handleChatLLMThreadConfigUpdate)
		api.GET("/chats/:id/messages", s.AuthMiddleware(), s.handleChatMessages)
		api.POST("/chats/:id/messages", s.AuthMiddleware(), s.handleChatSend)
		api.POST("/chats/:id/messages/:messageId/retry", s.AuthMiddleware(), s.handleChatRetry)
		api.GET("/chats/:id/messages/:messageId/markdown", s.AuthMiddleware(), s.handleChatSharedMarkdown)
		api.DELETE("/chats/:id/messages/:messageId", s.AuthMiddleware(), s.handleChatDelete)
		api.GET("/system-agent", s.AuthMiddleware(), s.handleSystemAgentStatus)
	}
}
