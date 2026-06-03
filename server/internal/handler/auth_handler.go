package handler

import (
	"errors"
	"net/http"

	"music-room/internal/middleware"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type RegisterRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if !middleware.BindAndValidate(c, &req) {
		return
	}

	if err := h.auth.Register(c.Request.Context(), req.Email, req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrEmailInUse):
			c.JSON(http.StatusConflict, gin.H{"error": "email already in use", "code": "EMAIL_IN_USE"})
		case errors.Is(err, service.ErrInvalidEmail):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid email address", "code": "INVALID_EMAIL"})
		case errors.Is(err, service.ErrWeakPassword):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "password must be at least 8 characters", "code": "WEAK_PASSWORD"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed", "code": "INTERNAL_ERROR"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "registration successful, check your email to verify your account"})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required", "code": "INVALID_REQUEST"})
		return
	}

	if err := h.auth.VerifyEmail(c.Request.Context(), token); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired verification token", "code": "INVALID_TOKEN"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed", "code": "INTERNAL_ERROR"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email verified successfully"})
}

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req ResendVerificationRequest
	if !middleware.BindAndValidate(c, &req) {
		return
	}

	if err := h.auth.ResendVerification(c.Request.Context(), req.Email); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidEmail):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid email address", "code": "INVALID_EMAIL"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed", "code": "INTERNAL_ERROR"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "if that email is registered and unverified, a new verification link has been sent"})
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !middleware.BindAndValidate(c, &req) {
		return
	}

	if err := h.auth.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidEmail):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid email address", "code": "INVALID_EMAIL"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed", "code": "INTERNAL_ERROR"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "if that email is registered, a reset link has been sent"})
}

type ResetPasswordRequest struct {
	Token    string `json:"token"    binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !middleware.BindAndValidate(c, &req) {
		return
	}

	if err := h.auth.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reset token", "code": "INVALID_TOKEN"})
		case errors.Is(err, service.ErrTokenExpired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "reset link has expired", "code": "TOKEN_EXPIRED"})
		case errors.Is(err, service.ErrTokenUsed):
			c.JSON(http.StatusBadRequest, gin.H{"error": "reset link has already been used", "code": "TOKEN_USED"})
		case errors.Is(err, service.ErrWeakPassword):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "password must be at least 8 characters", "code": "WEAK_PASSWORD"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "password reset failed", "code": "INTERNAL_ERROR"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset successful"})
}
