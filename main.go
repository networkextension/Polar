package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// ============ 数据模型 ============

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

// ============ PostgreSQL ============

var db *sql.DB

var errEmailExists = errors.New("email already exists")

// ============ Session 管理 ============

const (
	SessionCookieName = "session_id"
	SessionDuration   = 24 * time.Hour
)

func initDB() {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://gin_tester:test123456@localhost:5432/gin_auth?sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("ping postgres: %v", err)
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
		log.Fatalf("init schema: %v", err)
	}
}

// 生成随机 Session ID
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// 创建 Session
func createSession(user *User) (string, error) {
	sessionID := generateSessionID()
	session := &Session{
		ID:        sessionID,
		UserID:    user.ID,
		Username:  user.Username,
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	_, err := db.Exec(
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

// 获取 Session
func getSession(sessionID string) *Session {
	var session Session
	err := db.QueryRow(
		`SELECT id, user_id, username, expires_at FROM sessions WHERE id = $1`,
		sessionID,
	).Scan(&session.ID, &session.UserID, &session.Username, &session.ExpiresAt)
	if err != nil {
		return nil
	}

	if time.Now().After(session.ExpiresAt) {
		_ = deleteSession(sessionID)
		return nil
	}
	return &session
}

// 删除 Session (登出)
func deleteSession(sessionID string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}

// 清理过期 Session
func cleanupSessions() {
	for {
		time.Sleep(1 * time.Hour)
		_, _ = db.Exec(`DELETE FROM sessions WHERE expires_at < NOW()`)
	}
}

// ============ 密码处理 ============

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func getUserByEmail(email string) (*User, error) {
	var user User
	err := db.QueryRow(
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

func createUser(user *User) error {
	_, err := db.Exec(
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

// ============ Gin 中间件 ============

// AuthMiddleware 验证登录状态
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(SessionCookieName)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		session := getSession(sessionID)
		if session == nil {
			c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", session.UserID)
		c.Set("username", session.Username)
		c.Set("session", session)
		c.Next()
	}
}

// GuestMiddleware 阻止已登录用户访问登录/注册页
func GuestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(SessionCookieName)
		if err == nil {
			session := getSession(sessionID)
			if session != nil {
				c.Redirect(http.StatusFound, "/dashboard")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// ============ HTML 模板 ============

const layoutTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Gin Auth Demo</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
            width: 100%;
            max-width: 450px;
            animation: slideUp 0.5s ease;
        }
        @keyframes slideUp {
            from { opacity: 0; transform: translateY(30px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px 30px;
            text-align: center;
        }
        .header h1 { font-size: 28px; margin-bottom: 10px; }
        .header p { opacity: 0.9; font-size: 14px; }
        .form-container { padding: 40px 30px; }
        .form-group { margin-bottom: 20px; }
        .form-group label {
            display: block;
            margin-bottom: 8px;
            color: #333;
            font-weight: 500;
            font-size: 14px;
        }
        .form-group input {
            width: 100%;
            padding: 12px 15px;
            border: 2px solid #e0e0e0;
            border-radius: 10px;
            font-size: 15px;
            transition: all 0.3s;
        }
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
        }
        .btn {
            width: 100%;
            padding: 14px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 10px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(102, 126, 234, 0.3);
        }
        .btn:active { transform: translateY(0); }
        .links {
            text-align: center;
            margin-top: 20px;
            color: #666;
            font-size: 14px;
        }
        .links a {
            color: #667eea;
            text-decoration: none;
            font-weight: 600;
        }
        .links a:hover { text-decoration: underline; }
        .alert {
            padding: 12px 15px;
            border-radius: 8px;
            margin-bottom: 20px;
            font-size: 14px;
            display: none;
        }
        .alert.error {
            background: #fee;
            color: #c33;
            border: 1px solid #fcc;
            display: block;
        }
        .alert.success {
            background: #efe;
            color: #3c3;
            border: 1px solid #cfc;
            display: block;
        }
        .dashboard {
            max-width: 800px;
            width: 100%;
        }
        .nav {
            background: white;
            padding: 20px 30px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            border-radius: 15px;
            margin-bottom: 30px;
        }
        .nav-brand {
            font-size: 24px;
            font-weight: bold;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .nav-user {
            display: flex;
            align-items: center;
            gap: 15px;
        }
        .nav-user span { color: #666; }
        .btn-logout {
            padding: 8px 20px;
            background: #ff4757;
            color: white;
            border: none;
            border-radius: 20px;
            cursor: pointer;
            font-size: 14px;
            transition: all 0.3s;
        }
        .btn-logout:hover {
            background: #ee3742;
            transform: scale(1.05);
        }
        .card {
            background: white;
            border-radius: 15px;
            padding: 30px;
            box-shadow: 0 5px 20px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        .card h2 {
            color: #333;
            margin-bottom: 15px;
            font-size: 20px;
        }
        .card p { color: #666; line-height: 1.6; }
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-top: 20px;
        }
        .info-item {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 10px;
            text-align: center;
        }
        .info-item h3 {
            color: #667eea;
            font-size: 24px;
            margin-bottom: 5px;
        }
        .info-item p { color: #999; font-size: 14px; }
    </style>
</head>
<body>
    {{template "content" .}}
</body>
</html>`

const loginTemplate = `{{define "content"}}
<div class="container">
    <div class="header">
        <h1>👋 欢迎回来</h1>
        <p>登录您的账户以继续</p>
    </div>
    <div class="form-container">
        <div id="alert" class="alert"></div>
        <form id="loginForm">
            <div class="form-group">
                <label>邮箱地址</label>
                <input type="email" name="email" required placeholder="your@email.com">
            </div>
            <div class="form-group">
                <label>密码</label>
                <input type="password" name="password" required placeholder="••••••••">
            </div>
            <button type="submit" class="btn">登录</button>
        </form>
        <div class="links">
            还没有账户？ <a href="/register">立即注册</a>
        </div>
    </div>
</div>

<script>
document.getElementById('loginForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const formData = new FormData(e.target);
    const data = Object.fromEntries(formData);
    
    try {
        const res = await fetch('/api/login', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(data)
        });
        const result = await res.json();
        
        if (res.ok) {
            showAlert('success', '登录成功！正在跳转...');
            setTimeout(() => window.location.href = '/dashboard', 500);
        } else {
            showAlert('error', result.error || '登录失败');
        }
    } catch (err) {
        showAlert('error', '网络错误，请重试');
    }
});

function showAlert(type, msg) {
    const alert = document.getElementById('alert');
    alert.className = 'alert ' + type;
    alert.textContent = msg;
}
</script>
{{end}}`

const registerTemplate = `{{define "content"}}
<div class="container">
    <div class="header">
        <h1>🚀 创建账户</h1>
        <p>开始您的旅程</p>
    </div>
    <div class="form-container">
        <div id="alert" class="alert"></div>
        <form id="registerForm">
            <div class="form-group">
                <label>用户名</label>
                <input type="text" name="username" required placeholder="johndoe" minlength="3">
            </div>
            <div class="form-group">
                <label>邮箱地址</label>
                <input type="email" name="email" required placeholder="your@email.com">
            </div>
            <div class="form-group">
                <label>密码</label>
                <input type="password" name="password" required placeholder="至少6位字符" minlength="6">
            </div>
            <button type="submit" class="btn">注册</button>
        </form>
        <div class="links">
            已有账户？ <a href="/login">立即登录</a>
        </div>
    </div>
</div>

<script>
document.getElementById('registerForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const formData = new FormData(e.target);
    const data = Object.fromEntries(formData);
    
    try {
        const res = await fetch('/api/register', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(data)
        });
        const result = await res.json();
        
        if (res.ok) {
            showAlert('success', '注册成功！正在跳转...');
            setTimeout(() => window.location.href = '/dashboard', 500);
        } else {
            showAlert('error', result.error || '注册失败');
        }
    } catch (err) {
        showAlert('error', '网络错误，请重试');
    }
});

function showAlert(type, msg) {
    const alert = document.getElementById('alert');
    alert.className = 'alert ' + type;
    alert.textContent = msg;
}
</script>
{{end}}`

const dashboardTemplate = `{{define "content"}}
<div class="dashboard">
    <div class="nav">
        <div class="nav-brand">🔐 Gin Auth</div>
        <div class="nav-user">
            <span>👤 {{.Username}}</span>
            <button class="btn-logout" onclick="logout()">退出登录</button>
        </div>
    </div>
    
    <div class="card">
        <h2>欢迎回来，{{.Username}}！</h2>
        <p>您已成功登录系统。这是一个基于 Gin Framework 的 Session 认证示例应用。</p>
        
        <div class="info-grid">
            <div class="info-item">
                <h3>🔒</h3>
                <p>安全认证</p>
            </div>
            <div class="info-item">
                <h3>⚡</h3>
                <p>高性能</p>
            </div>
            <div class="info-item">
                <h3>🚀</h3>
                <p>现代化</p>
            </div>
        </div>
    </div>
    
    <div class="card">
        <h2>会话信息</h2>
        <p><strong>用户ID:</strong> {{.UserID}}</p>
        <p style="margin-top: 10px;"><strong>登录时间:</strong> {{.LoginTime}}</p>
    </div>
</div>

<script>
async function logout() {
    if (!confirm('确定要退出登录吗？')) return;
    
    try {
        const res = await fetch('/api/logout', {method: 'POST'});
        if (res.ok) {
            window.location.href = '/login';
        }
    } catch (err) {
        alert('退出失败，请重试');
    }
}
</script>
{{end}}`

// ============ 路由处理 ============

func main() {
	r := gin.Default()

	initDB()
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close db: %v", err)
		}
	}()

	// 启动 Session 清理协程
	go cleanupSessions()

	// 加载模板
	tmpl := template.Must(template.New("layout").Parse(layoutTemplate))
	template.Must(tmpl.New("login").Parse(loginTemplate))
	template.Must(tmpl.New("register").Parse(registerTemplate))
	template.Must(tmpl.New("dashboard").Parse(dashboardTemplate))
	r.SetHTMLTemplate(tmpl)

	// 静态页面路由
	r.GET("/", func(c *gin.Context) {
		sessionID, _ := c.Cookie(SessionCookieName)
		if sessionID != "" && getSession(sessionID) != nil {
			c.Redirect(http.StatusFound, "/dashboard")
			return
		}
		c.Redirect(http.StatusFound, "/login")
	})

	r.GET("/login", GuestMiddleware(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "login", gin.H{"Title": "登录"})
	})

	r.GET("/register", GuestMiddleware(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "register", gin.H{"Title": "注册"})
	})

	r.GET("/dashboard", AuthMiddleware(), func(c *gin.Context) {
		username, _ := c.Get("username")
		userID, _ := c.Get("user_id")
		c.HTML(http.StatusOK, "dashboard", gin.H{
			"Title":     "控制台",
			"Username":  username,
			"UserID":    userID,
			"LoginTime": time.Now().Format("2006-01-02 15:04:05"),
		})
	})

	// API 路由
	api := r.Group("/api")
	{
		api.POST("/register", handleRegister)
		api.POST("/login", handleLogin)
		api.POST("/logout", handleLogout)
		api.GET("/me", AuthMiddleware(), handleMe)
	}

	r.Run(":8080")
}

// 注册处理
func handleRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	// 检查邮箱是否已存在
	existingUser, err := getUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被注册"})
		return
	}

	// 哈希密码
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	// 创建用户
	user := &User{
		ID:        generateSessionID()[:16],
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword,
		CreatedAt: time.Now(),
	}
	if err := createUser(user); err != nil {
		if errors.Is(err, errEmailExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被注册"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	// 创建 Session
	sessionID, err := createSession(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.SetCookie(SessionCookieName, sessionID, int(SessionDuration.Seconds()), "/", "", false, true)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "注册成功",
		"user_id":  user.ID,
		"username": user.Username,
	})
}

// 登录处理
func handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	user, err := getUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if user == nil || !checkPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}

	// 创建 Session
	sessionID, err := createSession(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.SetCookie(SessionCookieName, sessionID, int(SessionDuration.Seconds()), "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message":  "登录成功",
		"user_id":  user.ID,
		"username": user.Username,
	})
}

// 登出处理
func handleLogout(c *gin.Context) {
	sessionID, err := c.Cookie(SessionCookieName)
	if err == nil {
		_ = deleteSession(sessionID)
	}

	// 清除 Cookie
	c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "已成功退出登录"})
}

// 获取当前用户信息
func handleMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
	})
}
