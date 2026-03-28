package dock

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultLang = "en"
	langZhCN    = "zh-CN"
	langCtxKey  = "lang"
)

var messageCatalog = map[string]map[string]string{
	"en": {
		"common.invalid_input":            "Invalid input",
		"common.server_error":             "Server error",
		"auth.unauthorized":               "Not logged in or session expired",
		"auth.forbidden":                  "Insufficient permissions",
		"auth.already_logged_in":          "Already logged in",
		"auth.email_registered":           "This email is already registered",
		"auth.invalid_credentials":        "Incorrect email or password",
		"auth.register_success":           "Registration successful",
		"auth.login_success":              "Login successful",
		"auth.logout_success":             "Logged out successfully",
		"email.service_unavailable":       "Email service is not configured",
		"email.send_failed":               "Failed to send verification email",
		"email.verification_sent":         "Verification email sent",
		"email.already_verified":          "Your email is already verified",
		"email.invalid_token":             "Invalid or expired verification link",
		"email.verify_success":            "Email verified successfully",
		"passkey.create_failed":           "Failed to create Passkey",
		"passkey.session_missing":         "Missing session information",
		"passkey.session_expired":         "Session expired, please try again",
		"passkey.verify_failed":           "Passkey verification failed",
		"passkey.save_failed":             "Failed to save Passkey",
		"passkey.update_failed":           "Failed to update Passkey",
		"passkey.bound_success":           "Passkey bound successfully",
		"passkey.deleted_success":         "Passkey deleted",
		"passkey.invalid_credential":      "Invalid Passkey",
		"passkey.delete_failed":           "Failed to delete Passkey",
		"passkey.not_found":               "Passkey not found",
		"passkey.user_missing_or_unbound": "User not found or no Passkey bound",
	},
	"zh-CN": {
		"common.invalid_input":            "无效的输入数据",
		"common.server_error":             "服务器错误",
		"auth.unauthorized":               "未登录或会话已失效",
		"auth.forbidden":                  "权限不足",
		"auth.already_logged_in":          "当前已登录",
		"auth.email_registered":           "该邮箱已被注册",
		"auth.invalid_credentials":        "邮箱或密码错误",
		"auth.register_success":           "注册成功",
		"auth.login_success":              "登录成功",
		"auth.logout_success":             "已成功退出登录",
		"email.service_unavailable":       "邮件服务未配置",
		"email.send_failed":               "发送验证邮件失败",
		"email.verification_sent":         "验证邮件已发送",
		"email.already_verified":          "你的邮箱已经验证过了",
		"email.invalid_token":             "验证链接无效或已过期",
		"email.verify_success":            "邮箱验证成功",
		"passkey.create_failed":           "创建 Passkey 失败",
		"passkey.session_missing":         "缺少会话信息",
		"passkey.session_expired":         "会话已过期，请重试",
		"passkey.verify_failed":           "Passkey 校验失败",
		"passkey.save_failed":             "保存 Passkey 失败",
		"passkey.update_failed":           "更新 Passkey 失败",
		"passkey.bound_success":           "Passkey 绑定成功",
		"passkey.deleted_success":         "Passkey 已删除",
		"passkey.invalid_credential":      "无效的 Passkey",
		"passkey.delete_failed":           "删除 Passkey 失败",
		"passkey.not_found":               "Passkey 不存在",
		"passkey.user_missing_or_unbound": "用户不存在或未绑定 Passkey",
	},
}

func normalizeLang(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	switch {
	case value == "", value == "*":
		return defaultLang
	case strings.HasPrefix(value, "zh"):
		return langZhCN
	default:
		return defaultLang
	}
}

func parseAcceptLanguage(header string) string {
	if header == "" {
		return defaultLang
	}
	parts := strings.Split(header, ",")
	for _, part := range parts {
		token := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if token == "" {
			continue
		}
		return normalizeLang(token)
	}
	return defaultLang
}

func detectRequestLang(r *http.Request) string {
	if r == nil {
		return defaultLang
	}
	if explicit := strings.TrimSpace(r.Header.Get("X-Language")); explicit != "" {
		return normalizeLang(explicit)
	}
	return parseAcceptLanguage(r.Header.Get("Accept-Language"))
}

func (s *Server) LanguageMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := detectRequestLang(c.Request)
		c.Set(langCtxKey, lang)
		c.Header("Content-Language", lang)
		c.Header("Vary", "Accept-Language, X-Language")
		c.Next()
	}
}

func requestLang(c *gin.Context) string {
	if c == nil {
		return defaultLang
	}
	if value, ok := c.Get(langCtxKey); ok {
		if lang, ok := value.(string); ok && lang != "" {
			return lang
		}
	}
	return defaultLang
}

func tr(c *gin.Context, key string) string {
	lang := requestLang(c)
	if locale, ok := messageCatalog[lang]; ok {
		if msg, ok := locale[key]; ok && msg != "" {
			return msg
		}
	}
	if fallback, ok := messageCatalog[defaultLang][key]; ok && fallback != "" {
		return fallback
	}
	return key
}

func jsonError(c *gin.Context, status int, key string) {
	c.JSON(status, gin.H{"error": tr(c, key)})
}

func jsonMessage(c *gin.Context, status int, key string, extra gin.H) {
	payload := gin.H{"message": tr(c, key)}
	for k, v := range extra {
		payload[k] = v
	}
	c.JSON(status, payload)
}
