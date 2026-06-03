package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewCORS returns a CORS middleware that whitelists the origins supplied in
// allowedOriginsEnv (comma-separated, e.g. "https://app.example.com,https://admin.example.com").
// An empty string means no origins are whitelisted — the middleware still runs
// but sets no CORS headers, so browsers will block all cross-origin requests.
func NewCORS(allowedOriginsEnv string) gin.HandlerFunc {
	origins := parseOrigins(allowedOriginsEnv)

	if len(origins) == 0 {
		// No origins configured: return a no-op so gin-contrib/cors doesn't
		// panic, but set no headers so the browser blocks cross-origin requests.
		return func(c *gin.Context) { c.Next() }
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}

func parseOrigins(raw string) []string {
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}
