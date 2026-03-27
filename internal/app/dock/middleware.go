package dock

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func isAPIRequest(c *gin.Context) bool {
	return strings.HasPrefix(c.Request.URL.Path, "/api/")
}

func clearSessionCookie(c *gin.Context) {
	c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)
}

func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(SessionCookieName)
		if err != nil {
			if isAPIRequest(c) {
				jsonError(c, http.StatusUnauthorized, "auth.unauthorized")
			} else {
				c.Redirect(http.StatusFound, "/login")
			}
			c.Abort()
			return
		}

		session := s.getSession(sessionID)
		if session == nil {
			clearSessionCookie(c)
			if isAPIRequest(c) {
				jsonError(c, http.StatusUnauthorized, "auth.unauthorized")
			} else {
				c.Redirect(http.StatusFound, "/login")
			}
			c.Abort()
			return
		}

		c.Set("user_id", session.UserID)
		c.Set("username", session.Username)
		if session.Role == "" {
			c.Set("role", "user")
		} else {
			c.Set("role", session.Role)
		}
		c.Set("session", session)
		c.Next()
	}
}

func (s *Server) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if roleStr, ok := role.(string); !ok || roleStr != "admin" {
			jsonError(c, http.StatusForbidden, "auth.forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) GuestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(SessionCookieName)
		if err == nil {
			session := s.getSession(sessionID)
			if session != nil {
				if isAPIRequest(c) {
					jsonError(c, http.StatusConflict, "auth.already_logged_in")
				} else {
					c.Redirect(http.StatusFound, "/dashboard")
				}
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
