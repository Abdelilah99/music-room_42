package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// NewLogger returns a Gin middleware that emits one structured JSON log line
// per request via slog. 4xx responses are logged at WARN, 5xx at ERROR,
// everything else at INFO.
func NewLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)

		userID, _ := c.Get("user_id")
		userIDStr, _ := userID.(string)

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.FullPath()),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("user_id", userIDStr),
			slog.String("platform", c.GetHeader("X-Platform")),
			slog.String("device_model", c.GetHeader("X-Device-Model")),
			slog.String("app_version", c.GetHeader("X-App-Version")),
		}

		ctx := c.Request.Context()
		switch {
		case status >= 500:
			slog.LogAttrs(ctx, slog.LevelError, "request", attrs...)
		case status >= 400:
			slog.LogAttrs(ctx, slog.LevelWarn, "request", attrs...)
		default:
			slog.LogAttrs(ctx, slog.LevelInfo, "request", attrs...)
		}
	}
}
