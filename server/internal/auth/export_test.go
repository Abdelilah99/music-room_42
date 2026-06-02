package auth

import (
	"context"
	"music-room/internal/model"
	"music-room/internal/repository"
)

func (h *Handler) TestGenerateTokenPair(ctx context.Context, user *model.User) (string, string, error) {
	return h.generateTokenPair(ctx, user)
}

// TestGoogleClaims exposes the unexported googleClaims type for use in tests.
type TestGoogleClaims = googleClaims

// NewGoogleHandlerForTest creates a GoogleHandler with an injected token verifier,
// allowing tests to bypass the real Google API call.
func NewGoogleHandlerForTest(
	oauthRepo repository.OAuthRepository,
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	jwtService *JWTService,
	verifier func(ctx context.Context, idToken string) (*googleClaims, error),
) *GoogleHandler {
	h := NewGoogleHandler(oauthRepo, userRepo, tokenRepo, jwtService)
	h.verifier = verifier
	return h
}
