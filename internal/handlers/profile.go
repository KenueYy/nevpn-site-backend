package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
	"github.com/KenueYy/nevpn-site-backend/internal/db"
	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func userRole(cfg *config.Config, email string) string {
	if cfg.IsAdmin(email) {
		return "admin"
	}
	return "user"
}

func Profile(c *gin.Context) {
	logger := c.MustGet("logger").(*slog.Logger)
	cfg := c.MustGet("cfg").(*config.Config)

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var user models.User
	if err := db.DB.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		logger.Error("get profile failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"role":       userRole(cfg, user.Email),
		"created_at": user.CreatedAt,
	})
}

// Me is an alias for Profile.
func Me(c *gin.Context) {
	Profile(c)
}
