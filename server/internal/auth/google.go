package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"music-room/internal/middleware"
	"music-room/internal/model"
	"music-room/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	ErrInvalidGoogleToken = errors.New("invalid or expired Google ID token")
)

type googleClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Aud           string `json:"aud"`
	Error         string `json:"error"`
}

func verifyGoogleIDToken(ctx context.Context, idToken string) (*googleClaims, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")

	url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var claims googleClaims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, err
	}

	if claims.Error != "" {
		return nil, ErrInvalidGoogleToken
	}

	if clientID != "" && claims.Aud != clientID {
		return nil, ErrInvalidGoogleToken
	}

	if claims.EmailVerified != "true" {
		return nil, ErrInvalidGoogleToken
	}

	if claims.Sub == "" || claims.Email == "" {
		return nil, ErrInvalidGoogleToken
	}

	return &claims, nil
}

type tokenVerifierFunc func(ctx context.Context, idToken string) (*googleClaims, error)

type googleSignInRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

type GoogleHandler struct {
	oauthRepo repository.OAuthRepository
	userRepo  repository.UserRepository
	tokenRepo repository.RefreshTokenRepository
	jwt       *JWTService
	verifier  tokenVerifierFunc
}

func NewGoogleHandler(
	oauthRepo repository.OAuthRepository,
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	jwtService *JWTService,
) *GoogleHandler {
	return &GoogleHandler{
		oauthRepo: oauthRepo,
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		jwt:       jwtService,
		verifier:  verifyGoogleIDToken,
	}
}

// SignIn godoc
// @Summary      Sign in or register with Google
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body googleSignInRequest true "Google ID token"
// @Success      200 {object} tokenResponse
// @Failure      400 {object} errorResponse
// @Failure      401 {object} errorResponse "Invalid Google ID token"
// @Failure      500 {object} errorResponse
// @Router       /auth/google [post]
func (h *GoogleHandler) SignIn(c *gin.Context) {
	var body struct {
		IDToken string `json:"id_token" binding:"required"`
	}
	if !middleware.BindAndValidate(c, &body) {
		return
	}

	claims, err := h.verifier(c.Request.Context(), body.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired Google ID token"})
		return
	}

	// Check if this Google account is already linked
	provider, err := h.oauthRepo.GetProviderByProviderID(c.Request.Context(), "google", claims.Sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	var user *model.User

	if provider != nil {
		user, err = h.userRepo.GetByID(c.Request.Context(), provider.UserID)
		if err != nil || user == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
			return
		}
	} else {
		// Check if a user with this email already exists (account linking)
		existing, err := h.userRepo.GetByEmail(c.Request.Context(), claims.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
			return
		}

		if existing != nil {
			user = existing
		} else {
			// Brand new user - create without password, pre-verified
			user, err = h.oauthRepo.CreateOAuthUser(c.Request.Context(), claims.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
				return
			}
		}

		if err := h.oauthRepo.CreateProvider(c.Request.Context(), user.ID, "google", claims.Sub); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
			return
		}
	}

	user.IsVerified = true

	accessToken, refreshStr, err := h.generateTokenPair(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshStr,
	})
}

// LinkGoogle godoc
// @Summary      Link a Google account to the authenticated user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body googleSignInRequest true "Google ID token"
// @Success      200 {object} messageResponse
// @Failure      400 {object} errorResponse
// @Failure      401 {object} errorResponse
// @Failure      409 {object} errorResponse "Google account already linked"
// @Failure      500 {object} errorResponse
// @Router       /auth/link/google [post]
func (h *GoogleHandler) LinkGoogle(c *gin.Context) {
	callerIDStr, _ := c.Get("user_id")
	callerID, err := uuid.Parse(callerIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var body struct {
		IDToken string `json:"id_token" binding:"required"`
	}
	if !middleware.BindAndValidate(c, &body) {
		return
	}

	claims, err := h.verifier(c.Request.Context(), body.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired Google ID token"})
		return
	}

	existing, err := h.oauthRepo.GetProviderByProviderID(c.Request.Context(), "google", claims.Sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}
	if existing != nil {
		if existing.UserID == callerID {
			c.JSON(http.StatusConflict, gin.H{"error": "Google account already linked to your account"})
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "Google account is already linked to another user"})
		}
		return
	}

	if err := h.oauthRepo.CreateProvider(c.Request.Context(), callerID, "google", claims.Sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Google account linked successfully"})
}

func (h *GoogleHandler) generateTokenPair(ctx context.Context, user *model.User) (string, string, error) {
	accessToken, err := h.jwt.GenerateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshStr, err := h.jwt.GenerateRefreshTokenString()
	if err != nil {
		return "", "", err
	}

	tokenHash := h.jwt.HashRefreshToken(refreshStr)
	expiresAt := time.Now().Add(h.jwt.GetRefreshTTL())

	if err := h.tokenRepo.Create(ctx, &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}); err != nil {
		return "", "", err
	}

	return accessToken, refreshStr, nil
}
