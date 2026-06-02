package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Middleware struct {
	jwtService *JWTService
}

func NewMiddleware(jwtService *JWTService) *Middleware {
	return &Middleware{jwtService: jwtService}
}

func (m *Middleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer <token>"})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := m.jwtService.ValidateAccessToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("subscription_tier", claims.SubscriptionTier)
		c.Next()
	}
}

func RequireOwnership(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxUserID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		paramUserID := c.Param(paramName)
		if paramUserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing ownership identifier in route"})
			c.Abort()
			return
		}

		if ctxUserID != paramUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to access this resource"})
			c.Abort()
			return
		}

		c.Next()
	}
}
