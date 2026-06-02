package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"music-room/internal/repository"
)

var (
	ErrEmailInUse   = errors.New("email already in use")
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("reset link has expired")
	ErrTokenUsed    = errors.New("reset link has already been used")
	ErrWeakPassword = errors.New("password must be at least 8 characters")
	ErrInvalidEmail = errors.New("invalid email address")
)

type AuthService struct {
	repo     *repository.AuthRepository
	emailSvc *EmailService
	appURL   string
}

func NewAuthService(repo *repository.AuthRepository, emailSvc *EmailService, appURL string) *AuthService {
	return &AuthService{repo: repo, emailSvc: emailSvc, appURL: appURL}
}

func normalizeEmail(raw string) (string, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", ErrInvalidEmail
	}
	normalized := strings.ToLower(addr.Address)
	if normalized != strings.ToLower(strings.TrimSpace(raw)) {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

func (s *AuthService) Register(ctx context.Context, email, password string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	if len(password) < 8 {
		return ErrWeakPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	token := uuid.New()
	_, err = s.repo.CreateUserWithVerification(ctx, normalized, string(hash), token)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailInUse
		}
		return fmt.Errorf("create user: %w", err)
	}

	link := fmt.Sprintf("%s/api/v1/auth/verify-email?token=%s", s.appURL, token)
	body := fmt.Sprintf("Hi,\n\nVerify your Music Room account by clicking the link below:\n\n%s\n\nIf you did not create an account, you can ignore this email.\n", link)
	if err := s.emailSvc.Send(normalized, "Verify your Music Room account", body); err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}

	return nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, tokenStr string) error {
	token, err := uuid.Parse(tokenStr)
	if err != nil {
		return ErrInvalidToken
	}

	userID, err := s.repo.GetAndDeleteEmailVerification(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidToken
		}
		return fmt.Errorf("verify email: %w", err)
	}

	if err := s.repo.VerifyUser(ctx, userID); err != nil {
		return fmt.Errorf("mark user verified: %w", err)
	}

	return nil
}

func (s *AuthService) ResendVerification(ctx context.Context, email string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return err
	}

	user, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get user: %w", err)
	}

	if user.IsVerified {
		return nil
	}

	if err := s.repo.DeleteEmailVerificationsForUser(ctx, user.ID); err != nil {
		return fmt.Errorf("clear old tokens: %w", err)
	}

	token := uuid.New()
	if err := s.repo.CreateEmailVerification(ctx, user.ID, token); err != nil {
		return fmt.Errorf("create verification token: %w", err)
	}

	link := fmt.Sprintf("%s/api/v1/auth/verify-email?token=%s", s.appURL, token)
	body := fmt.Sprintf("Hi,\n\nVerify your Music Room account by clicking the link below:\n\n%s\n\nIf you did not create an account, you can ignore this email.\n", link)
	if err := s.emailSvc.Send(normalized, "Verify your Music Room account", body); err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}

	return nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return err
	}

	user, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get user: %w", err)
	}

	token := uuid.New()
	expiresAt := time.Now().Add(time.Hour)
	if err := s.repo.CreatePasswordResetToken(ctx, user.ID, token, expiresAt); err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}

	link := fmt.Sprintf("%s/api/v1/auth/reset-password?token=%s", s.appURL, token)
	body := fmt.Sprintf("Hi,\n\nReset your Music Room password by clicking the link below:\n\n%s\n\nThis link expires in 1 hour. If you did not request a password reset, you can ignore this email.\n", link)
	if err := s.emailSvc.Send(normalized, "Reset your Music Room password", body); err != nil {
		return fmt.Errorf("send reset email: %w", err)
	}

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}

	token, err := uuid.Parse(tokenStr)
	if err != nil {
		return ErrInvalidToken
	}

	t, err := s.repo.GetPasswordResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidToken
		}
		return fmt.Errorf("get reset token: %w", err)
	}

	if time.Now().After(t.ExpiresAt) {
		return ErrTokenExpired
	}
	if t.UsedAt != nil {
		return ErrTokenUsed
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.ResetPassword(ctx, t.ID, t.UserID, string(hash)); err != nil {
		return fmt.Errorf("reset password: %w", err)
	}

	return nil
}
