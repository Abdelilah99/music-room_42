package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"music-room/internal/model"
	"music-room/internal/repository"
	"music-room/internal/service"
)

// compile-time interface check
var _ repository.ProfileRepository = (*mockProfileRepo)(nil)

type mockProfileRepo struct {
	getProfileByIDFn     func(ctx context.Context, id string) (*model.UserProfile, error)
	updateProfileFn      func(ctx context.Context, id string, req model.UpdateProfileRequest) error
	getFriendshipStatusFn func(ctx context.Context, userID, targetID string) (string, error)
	searchUsersFn        func(ctx context.Context, query string) ([]model.UserSearchResult, error)
}

func (m *mockProfileRepo) GetProfileByID(ctx context.Context, id string) (*model.UserProfile, error) {
	return m.getProfileByIDFn(ctx, id)
}
func (m *mockProfileRepo) UpdateProfile(ctx context.Context, id string, req model.UpdateProfileRequest) error {
	return m.updateProfileFn(ctx, id, req)
}
func (m *mockProfileRepo) GetFriendshipStatus(ctx context.Context, userID, targetID string) (string, error) {
	return m.getFriendshipStatusFn(ctx, userID, targetID)
}
func (m *mockProfileRepo) SearchUsers(ctx context.Context, query string) ([]model.UserSearchResult, error) {
	return m.searchUsersFn(ctx, query)
}

func newUserProfile(id string) *model.UserProfile {
	return &model.UserProfile{
		ID:          id,
		Email:       id + "@example.com",
		PublicInfo:  json.RawMessage(`{"name":"Test User"}`),
		FriendsInfo: json.RawMessage(`{"phone":"555-0100"}`),
		PrivateInfo: json.RawMessage(`{"dob":"1990-01-01"}`),
	}
}

// --- GetMyProfile ---

func TestProfileService_GetMyProfile_Success(t *testing.T) {
	myID := "user-1"
	want := newUserProfile(myID)

	repo := &mockProfileRepo{
		getProfileByIDFn: func(_ context.Context, id string) (*model.UserProfile, error) {
			if id != myID {
				t.Errorf("expected id %s, got %s", myID, id)
			}
			return want, nil
		},
	}

	svc := service.NewProfileService(repo)
	got, err := svc.GetMyProfile(context.Background(), myID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID || got.Email != want.Email {
		t.Errorf("expected profile %+v, got %+v", want, got)
	}
}

func TestProfileService_GetMyProfile_Error_Propagates(t *testing.T) {
	sentinel := errors.New("db error")

	repo := &mockProfileRepo{
		getProfileByIDFn: func(_ context.Context, _ string) (*model.UserProfile, error) {
			return nil, sentinel
		},
	}

	svc := service.NewProfileService(repo)
	_, err := svc.GetMyProfile(context.Background(), "user-1")
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

// --- UpdateMyProfile ---

func TestProfileService_UpdateMyProfile_Success(t *testing.T) {
	myID := "user-1"
	called := false
	pub := json.RawMessage(`{"name":"Updated"}`)

	repo := &mockProfileRepo{
		updateProfileFn: func(_ context.Context, id string, req model.UpdateProfileRequest) error {
			if id != myID {
				t.Errorf("expected id %s, got %s", myID, id)
			}
			called = true
			return nil
		},
	}

	svc := service.NewProfileService(repo)
	if err := svc.UpdateMyProfile(context.Background(), myID, model.UpdateProfileRequest{PublicInfo: &pub}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected UpdateProfile to be called")
	}
}

// --- SearchUsers ---

func TestProfileService_SearchUsers_ReturnsResults(t *testing.T) {
	results := []model.UserSearchResult{
		{ID: "u1", Email: "alice@example.com", Name: "Alice"},
		{ID: "u2", Email: "bob@example.com", Name: "Bob"},
	}

	repo := &mockProfileRepo{
		searchUsersFn: func(_ context.Context, q string) ([]model.UserSearchResult, error) {
			if q != "alice" {
				t.Errorf("unexpected query: %s", q)
			}
			return results, nil
		},
	}

	svc := service.NewProfileService(repo)
	got, err := svc.SearchUsers(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 results, got %d", len(got))
	}
}

// --- GetUserProfile ---

func TestProfileService_GetUserProfile_NonFriend_NoFriendsInfo(t *testing.T) {
	myID := "user-me"
	targetID := "user-target"
	target := newUserProfile(targetID)

	repo := &mockProfileRepo{
		getFriendshipStatusFn: func(_ context.Context, _, _ string) (string, error) {
			return "none", nil
		},
		getProfileByIDFn: func(_ context.Context, id string) (*model.UserProfile, error) {
			if id != targetID {
				t.Errorf("expected targetID %s, got %s", targetID, id)
			}
			return target, nil
		},
	}

	svc := service.NewProfileService(repo)
	got, err := svc.GetUserProfile(context.Background(), myID, targetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["friends_info"]; ok {
		t.Error("friends_info should not be present for a non-friend")
	}
	if got["id"] != targetID {
		t.Errorf("expected id %s, got %v", targetID, got["id"])
	}
}

func TestProfileService_GetUserProfile_Friend_IncludesFriendsInfo(t *testing.T) {
	myID := "user-me"
	targetID := "user-target"
	target := newUserProfile(targetID)

	repo := &mockProfileRepo{
		getFriendshipStatusFn: func(_ context.Context, _, _ string) (string, error) {
			return "accepted", nil
		},
		getProfileByIDFn: func(_ context.Context, _ string) (*model.UserProfile, error) {
			return target, nil
		},
	}

	svc := service.NewProfileService(repo)
	got, err := svc.GetUserProfile(context.Background(), myID, targetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["friends_info"]; !ok {
		t.Error("friends_info should be present for a friend")
	}
}

func TestProfileService_GetUserProfile_PendingRequest_NoFriendsInfo(t *testing.T) {
	target := newUserProfile("user-target")

	repo := &mockProfileRepo{
		getFriendshipStatusFn: func(_ context.Context, _, _ string) (string, error) {
			return "pending", nil
		},
		getProfileByIDFn: func(_ context.Context, _ string) (*model.UserProfile, error) {
			return target, nil
		},
	}

	svc := service.NewProfileService(repo)
	got, err := svc.GetUserProfile(context.Background(), "user-me", "user-target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["friends_info"]; ok {
		t.Error("friends_info should not be present for a pending request")
	}
}

func TestProfileService_GetUserProfile_StatusError_Propagates(t *testing.T) {
	sentinel := errors.New("db error")

	repo := &mockProfileRepo{
		getFriendshipStatusFn: func(_ context.Context, _, _ string) (string, error) {
			return "", sentinel
		},
	}

	svc := service.NewProfileService(repo)
	_, err := svc.GetUserProfile(context.Background(), "u1", "u2")
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestProfileService_GetUserProfile_ProfileError_Propagates(t *testing.T) {
	sentinel := errors.New("profile not found")

	repo := &mockProfileRepo{
		getFriendshipStatusFn: func(_ context.Context, _, _ string) (string, error) {
			return "none", nil
		},
		getProfileByIDFn: func(_ context.Context, _ string) (*model.UserProfile, error) {
			return nil, sentinel
		},
	}

	svc := service.NewProfileService(repo)
	_, err := svc.GetUserProfile(context.Background(), "u1", "u2")
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}
