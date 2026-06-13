package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"music-room/internal/model"
	"music-room/internal/repository"
	"music-room/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// compile-time checks
var _ repository.AuthRepository = (*mockAuthRepo)(nil)
var _ service.EmailSender = (*mockEmailSender)(nil)

// --- mock auth repository ---

type mockAuthRepo struct {
	createUserWithVerificationFn    func(ctx context.Context, email, passwordHash string, token uuid.UUID) (*model.User, error)
	getUserByEmailFn                func(ctx context.Context, email string) (*model.User, error)
	getAndDeleteEmailVerificationFn func(ctx context.Context, token uuid.UUID) (uuid.UUID, error)
	verifyUserFn                    func(ctx context.Context, userID uuid.UUID) error
	deleteEmailVerificationsForUserFn func(ctx context.Context, userID uuid.UUID) error
	createEmailVerificationFn       func(ctx context.Context, userID, token uuid.UUID) error
	createPasswordResetTokenFn      func(ctx context.Context, userID, token uuid.UUID, expiresAt time.Time) error
	getPasswordResetTokenFn         func(ctx context.Context, token uuid.UUID) (*model.PasswordResetToken, error)
	resetPasswordFn                 func(ctx context.Context, tokenID, userID uuid.UUID, newPasswordHash string) error
}

func (m *mockAuthRepo) CreateUserWithVerification(ctx context.Context, email, passwordHash string, token uuid.UUID) (*model.User, error) {
	return m.createUserWithVerificationFn(ctx, email, passwordHash, token)
}
func (m *mockAuthRepo) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return m.getUserByEmailFn(ctx, email)
}
func (m *mockAuthRepo) GetAndDeleteEmailVerification(ctx context.Context, token uuid.UUID) (uuid.UUID, error) {
	return m.getAndDeleteEmailVerificationFn(ctx, token)
}
func (m *mockAuthRepo) VerifyUser(ctx context.Context, userID uuid.UUID) error {
	return m.verifyUserFn(ctx, userID)
}
func (m *mockAuthRepo) DeleteEmailVerificationsForUser(ctx context.Context, userID uuid.UUID) error {
	return m.deleteEmailVerificationsForUserFn(ctx, userID)
}
func (m *mockAuthRepo) CreateEmailVerification(ctx context.Context, userID, token uuid.UUID) error {
	return m.createEmailVerificationFn(ctx, userID, token)
}
func (m *mockAuthRepo) CreatePasswordResetToken(ctx context.Context, userID, token uuid.UUID, expiresAt time.Time) error {
	return m.createPasswordResetTokenFn(ctx, userID, token, expiresAt)
}
func (m *mockAuthRepo) GetPasswordResetToken(ctx context.Context, token uuid.UUID) (*model.PasswordResetToken, error) {
	return m.getPasswordResetTokenFn(ctx, token)
}
func (m *mockAuthRepo) ResetPassword(ctx context.Context, tokenID, userID uuid.UUID, newPasswordHash string) error {
	return m.resetPasswordFn(ctx, tokenID, userID, newPasswordHash)
}

// --- mock email sender ---

type mockEmailSender struct {
	sendFn func(to, subject, body string) error
}

func (m *mockEmailSender) Send(to, subject, body string) error {
	return m.sendFn(to, subject, body)
}

func newEmailOK() *mockEmailSender {
	return &mockEmailSender{
		sendFn: func(_, _, _ string) error { return nil },
	}
}

// --- Register ---

func TestAuthService_Register_InvalidEmail_ReturnsError(t *testing.T) {
	svc := service.NewAuthService(&mockAuthRepo{}, newEmailOK(), "http://localhost")
	if err := svc.Register(context.Background(), "not-an-email", "password123"); !errors.Is(err, service.ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestAuthService_Register_WeakPassword_ReturnsError(t *testing.T) {
	svc := service.NewAuthService(&mockAuthRepo{}, newEmailOK(), "http://localhost")
	if err := svc.Register(context.Background(), "user@example.com", "short"); !errors.Is(err, service.ErrWeakPassword) {
		t.Errorf("expected ErrWeakPassword, got %v", err)
	}
}

func TestAuthService_Register_DuplicateEmail_ReturnsConflict(t *testing.T) {
	uniqueErr := &pgconn.PgError{Code: "23505"}
	repo := &mockAuthRepo{
		createUserWithVerificationFn: func(_ context.Context, _, _ string, _ uuid.UUID) (*model.User, error) {
			return nil, uniqueErr
		},
	}

	svc := service.NewAuthService(repo, newEmailOK(), "http://localhost")
	if err := svc.Register(context.Background(), "user@example.com", "password123"); !errors.Is(err, service.ErrEmailInUse) {
		t.Errorf("expected ErrEmailInUse, got %v", err)
	}
}

func TestAuthService_Register_Success_SendsEmail(t *testing.T) {
	userID := uuid.New()
	emailSent := false

	repo := &mockAuthRepo{
		createUserWithVerificationFn: func(_ context.Context, email, _ string, _ uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, Email: email}, nil
		},
	}
	emailSvc := &mockEmailSender{
		sendFn: func(to, subject, _ string) error {
			if to != "user@example.com" {
				t.Errorf("unexpected email recipient: %s", to)
			}
			emailSent = true
			return nil
		},
	}

	svc := service.NewAuthService(repo, emailSvc, "http://localhost")
	if err := svc.Register(context.Background(), "user@example.com", "password123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emailSent {
		t.Error("expected verification email to be sent")
	}
}

// --- VerifyEmail ---

func TestAuthService_VerifyEmail_InvalidToken_ReturnsError(t *testing.T) {
	svc := service.NewAuthService(&mockAuthRepo{}, newEmailOK(), "http://localhost")
	if err := svc.VerifyEmail(context.Background(), "not-a-uuid"); !errors.Is(err, service.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_VerifyEmail_UnknownToken_ReturnsError(t *testing.T) {
	repo := &mockAuthRepo{
		getAndDeleteEmailVerificationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return uuid.Nil, pgx.ErrNoRows
		},
	}

	svc := service.NewAuthService(repo, newEmailOK(), "http://localhost")
	if err := svc.VerifyEmail(context.Background(), uuid.New().String()); !errors.Is(err, service.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_VerifyEmail_Success(t *testing.T) {
	userID := uuid.New()
	verified := false

	repo := &mockAuthRepo{
		getAndDeleteEmailVerificationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return userID, nil
		},
		verifyUserFn: func(_ context.Context, uid uuid.UUID) error {
			if uid != userID {
				t.Errorf("unexpected userID: %s", uid)
			}
			verified = true
			return nil
		},
	}

	svc := service.NewAuthService(repo, newEmailOK(), "http://localhost")
	if err := svc.VerifyEmail(context.Background(), uuid.New().String()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !verified {
		t.Error("expected VerifyUser to be called")
	}
}

// --- ResendVerification ---

func TestAuthService_ResendVerification_InvalidEmail_ReturnsError(t *testing.T) {
	svc := service.NewAuthService(&mockAuthRepo{}, newEmailOK(), "http://localhost")
	if err := svc.ResendVerification(context.Background(), "bad-email"); !errors.Is(err, service.ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestAuthService_ResendVerification_UnknownEmail_Succeeds(t *testing.T) {
	// Should silently return nil to avoid leaking whether the email exists
	repo := &mockAuthRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewAuthService(repo, newEmailOK(), "http://localhost")
	if err := svc.ResendVerification(context.Background(), "unknown@example.com"); err != nil {
		t.Fatalf("expected nil for unknown email, got %v", err)
	}
}

func TestAuthService_ResendVerification_AlreadyVerified_Succeeds(t *testing.T) {
	repo := &mockAuthRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
			return &model.User{ID: uuid.New(), IsVerified: true}, nil
		},
	}

	svc := service.NewAuthService(repo, newEmailOK(), "http://localhost")
	if err := svc.ResendVerification(context.Background(), "verified@example.com"); err != nil {
		t.Fatalf("expected nil for already-verified user, got %v", err)
	}
}

func TestAuthService_ResendVerification_Success_SendsEmail(t *testing.T) {
	userID := uuid.New()
	emailSent := false

	repo := &mockAuthRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
			return &model.User{ID: userID, IsVerified: false}, nil
		},
		deleteEmailVerificationsForUserFn: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
		createEmailVerificationFn: func(_ context.Context, _, _ uuid.UUID) error {
			return nil
		},
	}
	emailSvc := &mockEmailSender{
		sendFn: func(_, _, _ string) error {
			emailSent = true
			return nil
		},
	}

	svc := service.NewAuthService(repo, emailSvc, "http://localhost")
	if err := svc.ResendVerification(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emailSent {
		t.Error("expected verification email to be re-sent")
	}
}

// --- ForgotPassword ---

func TestAuthService_ForgotPassword_InvalidEmail_ReturnsError(t *testing.T) {
	svc := service.NewAuthService(&mockAuthRepo{}, newEmailOK(), "http://localhost")
	if err := svc.ForgotPassword(context.Background(), "bad-email"); !errors.Is(err, service.ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestAuthService_ForgotPassword_UnknownEmail_Succeeds(t *testing.T) {
	repo := &mockAuthRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewAuthService(repo, newEmailOK(), "http://localhost")
	if err := svc.ForgotPassword(context.Background(), "unknown@example.com"); err != nil {
		t.Fatalf("expected nil for unknown email, got %v", err)
	}
}

func TestAuthService_ForgotPassword_Success_SendsResetEmail(t *testing.T) {
	userID := uuid.New()
	emailSent := false

	repo := &mockAuthRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
			return &model.User{ID: userID}, nil
		},
		createPasswordResetTokenFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ time.Time) error {
			return nil
		},
	}
	emailSvc := &mockEmailSender{
		sendFn: func(to, subject, _ string) error {
			if to != "user@example.com" {
				t.Errorf("unexpected recipient: %s", to)
			}
			emailSent = true
			return nil
		},
	}

	svc := service.NewAuthService(repo, emailSvc, "http://localhost")
	if err := svc.ForgotPassword(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emailSent {
		t.Error("expected reset email to be sent")
	}
}

// --- ResetPassword ---

func TestAuthService_ResetPassword_WeakPassword_ReturnsError(t *testing.T) {
	svc := service.NewAuthService(&mockAuthRepo{}, newEmailOK(), "http://localhost")
	if err := svc.ResetPassword(context.Background(), uuid.New().String(), "short"); !errors.Is(err, service.ErrWeakPassword) {
		t.Errorf("expected ErrWeakPassword, got %v", err)
	}
}

func TestAuthService_ResetPassword_InvalidToken_ReturnsError(t *testing.T) {
	svc := service.NewAuthService(&mockAuthRepo{}, newEmailOK(), "http://localhost")
	if err := svc.ResetPassword(context.Background(), "not-a-uuid", "password123"); !errors.Is(err, service.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_ResetPassword_UnknownToken_ReturnsError(t *testing.T) {
	repo := &mockAuthRepo{
		getPasswordResetTokenFn: func(_ context.Context, _ uuid.UUID) (*model.PasswordResetToken, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewAuthService(repo, newEmailOK(), "http://localhost")
	if err := svc.ResetPassword(context.Background(), uuid.New().String(), "password123"); !errors.Is(err, service.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_ResetPassword_ExpiredToken_ReturnsError(t *testing.T) {
	expired := time.Now().Add(-time.Hour)
	repo := &mockAuthRepo{
		getPasswordResetTokenFn: func(_ context.Context, _ uuid.UUID) (*model.PasswordResetToken, error) {
			return &model.PasswordResetToken{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				ExpiresAt: expired,
			}, nil
		},
	}

	svc := service.NewAuthService(repo, newEmailOK(), "http://localhost")
	if err := svc.ResetPassword(context.Background(), uuid.New().String(), "password123"); !errors.Is(err, service.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestAuthService_ResetPassword_AlreadyUsedToken_ReturnsError(t *testing.T) {
	usedAt := time.Now().Add(-time.Minute)
	repo := &mockAuthRepo{
		getPasswordResetTokenFn: func(_ context.Context, _ uuid.UUID) (*model.PasswordResetToken, error) {
			return &model.PasswordResetToken{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				ExpiresAt: time.Now().Add(time.Hour),
				UsedAt:    &usedAt,
			}, nil
		},
	}

	svc := service.NewAuthService(repo, newEmailOK(), "http://localhost")
	if err := svc.ResetPassword(context.Background(), uuid.New().String(), "password123"); !errors.Is(err, service.ErrTokenUsed) {
		t.Errorf("expected ErrTokenUsed, got %v", err)
	}
}

func TestAuthService_ResetPassword_Success(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	reset := false

	repo := &mockAuthRepo{
		getPasswordResetTokenFn: func(_ context.Context, _ uuid.UUID) (*model.PasswordResetToken, error) {
			return &model.PasswordResetToken{
				ID:        tokenID,
				UserID:    userID,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
		resetPasswordFn: func(_ context.Context, tID, uID uuid.UUID, _ string) error {
			if tID != tokenID || uID != userID {
				t.Errorf("unexpected IDs: token=%s user=%s", tID, uID)
			}
			reset = true
			return nil
		},
	}

	svc := service.NewAuthService(repo, newEmailOK(), "http://localhost")
	if err := svc.ResetPassword(context.Background(), uuid.New().String(), "newpassword123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reset {
		t.Error("expected ResetPassword to be called on repository")
	}
}
