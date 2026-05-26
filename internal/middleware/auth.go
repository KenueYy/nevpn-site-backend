package middleware

import (
	"net/http"
	"strings"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func RequireAuth(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	email := session.Get("email")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	c.Set("user_id", userID)
	c.Set("email", email)
	c.Next()
}

func RequireAdmin(c *gin.Context) {
	cfgAny, ok := c.Get("cfg")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "config missing"})
		c.Abort()
		return
	}
	cfg := cfgAny.(*config.Config)

	session := sessions.Default(c)
	emailVal := session.Get("email")
	email, _ := emailVal.(string)
	email = strings.ToLower(strings.TrimSpace(email))

	if !cfg.IsAdmin(email) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not admin"})
		c.Abort()
		return
	}

	c.Set("admin", true)
	c.Next()
}
