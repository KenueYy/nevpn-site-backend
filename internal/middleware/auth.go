package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func RequireAuth(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	c.Set("user_id", userID)
	c.Next()
}

func RequireAdmin(c *gin.Context) {
	session := sessions.Default(c)
	email := session.Get("email")
	if email != "kenueyy@gmail.com" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "you are not admin"})
		c.Abort()
		return
	}

	c.Set("admin", true)
}
