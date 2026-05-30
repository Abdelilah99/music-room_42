package auth

import (
	"context"
	"net/http"
	"time"
	"music-room/internal/model"
	"music-room/internal/repository"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	userRepo   repository.UserRepository
	tokenRepo  repository.RefreshTokenRepository
	jwtService *JWTService
}

func NewHandler(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	jwtService *JWTService,
) *Handler {
	return &Handler{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtService: jwtService,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	accessToken, refreshTokenStr, err := h.generateTokenPair(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshTokenStr,
	})
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash := h.jwtService.HashRefreshToken(req.RefreshToken)
	token, err := h.tokenRepo.GetByHash(c.Request.Context(), hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if token == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	if token.RevokedAt != nil {
		// Detect reuse, revoke all user tokens for security
		_ = h.tokenRepo.RevokeAllForUser(c.Request.Context(), token.UserID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
		return
	}

	if time.Now().After(token.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token has expired"})
		return
	}

	// Revoke current token
	err = h.tokenRepo.Revoke(c.Request.Context(), token.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), token.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	accessToken, newRefreshTokenStr, err := h.generateTokenPair(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshTokenStr,
	})
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash := h.jwtService.HashRefreshToken(req.RefreshToken)
	token, err := h.tokenRepo.GetByHash(c.Request.Context(), hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if token != nil {
		_ = h.tokenRepo.Revoke(c.Request.Context(), token.ID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *Handler) generateTokenPair(
	ctx context.Context,
	user *model.User,
) (string, string, error) {
	accessToken, err := h.jwtService.GenerateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshTokenStr, err := h.jwtService.GenerateRefreshTokenString()
	if err != nil {
		return "", "", err
	}

	tokenHash := h.jwtService.HashRefreshToken(refreshTokenStr)
	expiresAt := time.Now().Add(h.jwtService.GetRefreshTTL())

	tokenModel := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}

	err = h.tokenRepo.Create(ctx, tokenModel)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshTokenStr, nil
}
