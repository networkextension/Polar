package dock

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	db          *sql.DB
	router      *gin.Engine
	addr        string
	markdownDir string
}

func NewServer(cfg Config) (*Server, error) {
	db, err := openDB(cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	server := &Server{
		db:          db,
		addr:        cfg.Addr,
		markdownDir: cfg.MarkdownDir,
	}

	server.router = gin.Default()
	server.registerRoutes()

	go server.cleanupSessions()

	return server, nil
}

func (s *Server) Run() error {
	return s.router.Run(s.addr)
}

func (s *Server) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
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
		c.HTML(http.StatusOK, "dashboard", gin.H{
			"Title":     "控制台",
			"Username":  username,
			"UserID":    userID,
			"LoginTime": time.Now().Format("2006-01-02 15:04:05"),
		})
	})

	api := s.router.Group("/api")
	{
		api.POST("/register", s.handleRegister)
		api.POST("/login", s.handleLogin)
		api.POST("/logout", s.handleLogout)
		api.GET("/me", s.AuthMiddleware(), s.handleMe)
		api.POST("/markdown", s.AuthMiddleware(), s.handleMarkdownSubmit)
		api.GET("/markdown", s.AuthMiddleware(), s.handleMarkdownList)
		api.GET("/markdown/:id", s.AuthMiddleware(), s.handleMarkdownRead)
	}
}
