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
)

// compile-time interface check
var _ repository.FriendRepository = (*mockFriendRepo)(nil)

type mockFriendRepo struct {
	getExistingFn          func(ctx context.Context, userA, userB uuid.UUID) (*model.Friendship, error)
	createFn               func(ctx context.Context, requesterID, addresseeID uuid.UUID) (*model.Friendship, error)
	getByIDFn              func(ctx context.Context, id uuid.UUID) (*model.Friendship, error)
	acceptFn               func(ctx context.Context, id uuid.UUID) error
	deleteFn               func(ctx context.Context, id uuid.UUID) error
	listFriendsFn          func(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error)
	listIncomingFn         func(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error)
	listOutgoingFn         func(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error)
}

func (m *mockFriendRepo) GetExisting(ctx context.Context, userA, userB uuid.UUID) (*model.Friendship, error) {
	return m.getExistingFn(ctx, userA, userB)
}
func (m *mockFriendRepo) Create(ctx context.Context, requesterID, addresseeID uuid.UUID) (*model.Friendship, error) {
	return m.createFn(ctx, requesterID, addresseeID)
}
func (m *mockFriendRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Friendship, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockFriendRepo) Accept(ctx context.Context, id uuid.UUID) error {
	return m.acceptFn(ctx, id)
}
func (m *mockFriendRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}
func (m *mockFriendRepo) ListFriends(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error) {
	return m.listFriendsFn(ctx, userID)
}
func (m *mockFriendRepo) ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error) {
	return m.listIncomingFn(ctx, userID)
}
func (m *mockFriendRepo) ListOutgoingRequests(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error) {
	return m.listOutgoingFn(ctx, userID)
}

// helpers

func newFriendship(requesterID, addresseeID uuid.UUID, status string) *model.Friendship {
	return &model.Friendship{
		ID:          uuid.New(),
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Status:      status,
		CreatedAt:   time.Now(),
	}
}

// --- SendRequest ---

func TestFriendService_SendRequest_CannotFriendSelf(t *testing.T) {
	svc := service.NewFriendService(&mockFriendRepo{})
	id := uuid.New()
	_, err := svc.SendRequest(context.Background(), id, id)
	if !errors.Is(err, service.ErrCannotFriendSelf) {
		t.Errorf("expected ErrCannotFriendSelf, got %v", err)
	}
}

func TestFriendService_SendRequest_DuplicateRequest_ReturnsConflict(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	existing := newFriendship(requesterID, addresseeID, "pending")

	repo := &mockFriendRepo{
		getExistingFn: func(_ context.Context, _, _ uuid.UUID) (*model.Friendship, error) {
			return existing, nil
		},
	}

	svc := service.NewFriendService(repo)
	_, err := svc.SendRequest(context.Background(), requesterID, addresseeID)
	if !errors.Is(err, service.ErrFriendshipExists) {
		t.Errorf("expected ErrFriendshipExists, got %v", err)
	}
}

func TestFriendService_SendRequest_AlreadyFriends_ReturnsConflict(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	existing := newFriendship(requesterID, addresseeID, "accepted")

	repo := &mockFriendRepo{
		getExistingFn: func(_ context.Context, _, _ uuid.UUID) (*model.Friendship, error) {
			return existing, nil
		},
	}

	svc := service.NewFriendService(repo)
	_, err := svc.SendRequest(context.Background(), requesterID, addresseeID)
	if !errors.Is(err, service.ErrFriendshipExists) {
		t.Errorf("expected ErrFriendshipExists, got %v", err)
	}
}

func TestFriendService_SendRequest_Success(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	want := newFriendship(requesterID, addresseeID, "pending")

	repo := &mockFriendRepo{
		getExistingFn: func(_ context.Context, _, _ uuid.UUID) (*model.Friendship, error) {
			return nil, pgx.ErrNoRows
		},
		createFn: func(_ context.Context, rID, aID uuid.UUID) (*model.Friendship, error) {
			if rID != requesterID || aID != addresseeID {
				t.Errorf("unexpected create args: requester=%s addressee=%s", rID, aID)
			}
			return want, nil
		},
	}

	svc := service.NewFriendService(repo)
	got, err := svc.SendRequest(context.Background(), requesterID, addresseeID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected friendship ID %s, got %s", want.ID, got.ID)
	}
}

func TestFriendService_SendRequest_RepoError_Propagates(t *testing.T) {
	sentinel := errors.New("db error")
	repo := &mockFriendRepo{
		getExistingFn: func(_ context.Context, _, _ uuid.UUID) (*model.Friendship, error) {
			return nil, sentinel
		},
	}

	svc := service.NewFriendService(repo)
	_, err := svc.SendRequest(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

// --- AcceptRequest ---

func TestFriendService_AcceptRequest_NotFound(t *testing.T) {
	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewFriendService(repo)
	err := svc.AcceptRequest(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrFriendshipNotFound) {
		t.Errorf("expected ErrFriendshipNotFound, got %v", err)
	}
}

func TestFriendService_AcceptRequest_NotAddressee_Rejected(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	f := newFriendship(requesterID, addresseeID, "pending")

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
	}

	svc := service.NewFriendService(repo)
	// The requester calls Accept — should be rejected as only the addressee may accept
	err := svc.AcceptRequest(context.Background(), f.ID, requesterID)
	if !errors.Is(err, service.ErrNotAddresseeOp) {
		t.Errorf("expected ErrNotAddresseeOp, got %v", err)
	}
}

func TestFriendService_AcceptRequest_AlreadyAccepted_ReturnsNotPending(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	f := newFriendship(requesterID, addresseeID, "accepted")

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
	}

	svc := service.NewFriendService(repo)
	err := svc.AcceptRequest(context.Background(), f.ID, addresseeID)
	if !errors.Is(err, service.ErrRequestNotPending) {
		t.Errorf("expected ErrRequestNotPending, got %v", err)
	}
}

func TestFriendService_AcceptRequest_Success(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	f := newFriendship(requesterID, addresseeID, "pending")
	accepted := false

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
		acceptFn: func(_ context.Context, id uuid.UUID) error {
			if id != f.ID {
				t.Errorf("unexpected friendship ID: %s", id)
			}
			accepted = true
			return nil
		},
	}

	svc := service.NewFriendService(repo)
	if err := svc.AcceptRequest(context.Background(), f.ID, addresseeID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !accepted {
		t.Error("expected Accept to be called on repository")
	}
}

// --- RejectRequest ---

func TestFriendService_RejectRequest_NotFound(t *testing.T) {
	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewFriendService(repo)
	err := svc.RejectRequest(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrFriendshipNotFound) {
		t.Errorf("expected ErrFriendshipNotFound, got %v", err)
	}
}

func TestFriendService_RejectRequest_NotAddressee_Rejected(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	f := newFriendship(requesterID, addresseeID, "pending")

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
	}

	svc := service.NewFriendService(repo)
	err := svc.RejectRequest(context.Background(), f.ID, requesterID)
	if !errors.Is(err, service.ErrNotAddresseeOp) {
		t.Errorf("expected ErrNotAddresseeOp, got %v", err)
	}
}

func TestFriendService_RejectRequest_AlreadyAccepted_ReturnsNotPending(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	f := newFriendship(requesterID, addresseeID, "accepted")

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
	}

	svc := service.NewFriendService(repo)
	err := svc.RejectRequest(context.Background(), f.ID, addresseeID)
	if !errors.Is(err, service.ErrRequestNotPending) {
		t.Errorf("expected ErrRequestNotPending, got %v", err)
	}
}

func TestFriendService_RejectRequest_Success(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	f := newFriendship(requesterID, addresseeID, "pending")
	deleted := false

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			if id != f.ID {
				t.Errorf("unexpected friendship ID: %s", id)
			}
			deleted = true
			return nil
		},
	}

	svc := service.NewFriendService(repo)
	if err := svc.RejectRequest(context.Background(), f.ID, addresseeID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called on repository")
	}
}

// --- Unfriend ---

func TestFriendService_Unfriend_NotFound(t *testing.T) {
	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := service.NewFriendService(repo)
	err := svc.Unfriend(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrFriendshipNotFound) {
		t.Errorf("expected ErrFriendshipNotFound, got %v", err)
	}
}

func TestFriendService_Unfriend_NotParticipant_Rejected(t *testing.T) {
	f := newFriendship(uuid.New(), uuid.New(), "accepted")
	outsider := uuid.New()

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
	}

	svc := service.NewFriendService(repo)
	err := svc.Unfriend(context.Background(), f.ID, outsider)
	if !errors.Is(err, service.ErrNotParticipantOp) {
		t.Errorf("expected ErrNotParticipantOp, got %v", err)
	}
}

func TestFriendService_Unfriend_Pending_ByAddressee_Rejected(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	f := newFriendship(requesterID, addresseeID, "pending")

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
	}

	svc := service.NewFriendService(repo)
	// Addressee tries to cancel a pending request — only the requester can do this
	err := svc.Unfriend(context.Background(), f.ID, addresseeID)
	if !errors.Is(err, service.ErrNotParticipantOp) {
		t.Errorf("expected ErrNotParticipantOp, got %v", err)
	}
}

func TestFriendService_Unfriend_Pending_ByRequester_Succeeds(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	f := newFriendship(requesterID, addresseeID, "pending")
	deleted := false

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			if id != f.ID {
				t.Errorf("unexpected friendship ID: %s", id)
			}
			deleted = true
			return nil
		},
	}

	svc := service.NewFriendService(repo)
	if err := svc.Unfriend(context.Background(), f.ID, requesterID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called on repository")
	}
}

func TestFriendService_Unfriend_Accepted_ByRequester_Succeeds(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	f := newFriendship(requesterID, addresseeID, "accepted")
	deleted := false

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}

	svc := service.NewFriendService(repo)
	if err := svc.Unfriend(context.Background(), f.ID, requesterID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called on repository")
	}
}

func TestFriendService_Unfriend_Accepted_ByAddressee_Succeeds(t *testing.T) {
	requesterID := uuid.New()
	addresseeID := uuid.New()
	f := newFriendship(requesterID, addresseeID, "accepted")
	deleted := false

	repo := &mockFriendRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Friendship, error) {
			return f, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}

	svc := service.NewFriendService(repo)
	if err := svc.Unfriend(context.Background(), f.ID, addresseeID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called on repository")
	}
}

// --- List operations ---

func TestFriendService_ListFriends_ReturnsList(t *testing.T) {
	userID := uuid.New()
	entries := []model.FriendEntry{
		{FriendshipID: uuid.New(), UserID: uuid.New(), Email: "a@example.com"},
		{FriendshipID: uuid.New(), UserID: uuid.New(), Email: "b@example.com"},
	}

	repo := &mockFriendRepo{
		listFriendsFn: func(_ context.Context, uid uuid.UUID) ([]model.FriendEntry, error) {
			if uid != userID {
				t.Errorf("unexpected userID: %s", uid)
			}
			return entries, nil
		},
	}

	svc := service.NewFriendService(repo)
	got, err := svc.ListFriends(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(entries) {
		t.Errorf("expected %d entries, got %d", len(entries), len(got))
	}
}

func TestFriendService_ListIncomingRequests_ReturnsList(t *testing.T) {
	userID := uuid.New()
	entries := []model.FriendEntry{
		{FriendshipID: uuid.New(), UserID: uuid.New(), Email: "sender@example.com"},
	}

	repo := &mockFriendRepo{
		listIncomingFn: func(_ context.Context, uid uuid.UUID) ([]model.FriendEntry, error) {
			if uid != userID {
				t.Errorf("unexpected userID: %s", uid)
			}
			return entries, nil
		},
	}

	svc := service.NewFriendService(repo)
	got, err := svc.ListIncomingRequests(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 entry, got %d", len(got))
	}
}

func TestFriendService_ListOutgoingRequests_ReturnsList(t *testing.T) {
	userID := uuid.New()
	entries := []model.FriendEntry{
		{FriendshipID: uuid.New(), UserID: uuid.New(), Email: "target@example.com"},
	}

	repo := &mockFriendRepo{
		listOutgoingFn: func(_ context.Context, uid uuid.UUID) ([]model.FriendEntry, error) {
			if uid != userID {
				t.Errorf("unexpected userID: %s", uid)
			}
			return entries, nil
		},
	}

	svc := service.NewFriendService(repo)
	got, err := svc.ListOutgoingRequests(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 entry, got %d", len(got))
	}
}
