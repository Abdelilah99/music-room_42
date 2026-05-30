package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"
	"music-room/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

type Claims struct {
	UserID           string `json:"user_id"`
	Email            string `json:"email"`
	SubscriptionTier string `json:"subscription_tier"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJWTService() *JWTService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "development-default-secret-key"
	}

	accessTTLStr := os.Getenv("JWT_ACCESS_TTL")
	accessTTL := 15 * time.Minute
	if accessTTLStr != "" {
		if d, err := time.ParseDuration(accessTTLStr); err == nil {
			accessTTL = d
		}
	}

	refreshTTLStr := os.Getenv("JWT_REFRESH_TTL")
	refreshTTL := 7 * 24 * time.Hour
	if refreshTTLStr != "" {
		if d, err := time.ParseDuration(refreshTTLStr); err == nil {
			refreshTTL = d
		}
	}

	return &JWTService{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *JWTService) GenerateAccessToken(user *model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:           user.ID,
		Email:            user.Email,
		SubscriptionTier: user.SubscriptionTier,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTService) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *JWTService) GenerateRefreshTokenString() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *JWTService) HashRefreshToken(tokenStr string) string {
	h := sha256.New()
	h.Write([]byte(tokenStr))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *JWTService) GetRefreshTTL() time.Duration {
	return s.refreshTTL
}
