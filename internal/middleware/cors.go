package middleware

import (
	"net/http"
	"strings"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
	"github.com/gin-gonic/gin"
)

func CORS(cfg *config.Config) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(cfg.CorsOrigins))
	for _, o := range cfg.CorsOrigins {
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// CorsOriginAllowed exported for tests
func CorsOriginAllowed(cfg *config.Config, origin string) bool {
	for _, o := range cfg.CorsOrigins {
		if strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}
