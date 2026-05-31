package service

import (
	"context"

	"music-room/internal/model"
	"music-room/internal/repository"
)

type ProfileService interface {
	GetMyProfile(ctx context.Context, myID string) (*model.UserProfile, error)
	UpdateMyProfile(ctx context.Context, myID string, req model.UpdateProfileRequest) error
	GetUserProfile(ctx context.Context, myID, targetID string) (map[string]interface{}, error)
}

type profileService struct {
	repo repository.ProfileRepository
}

func NewProfileService(repo repository.ProfileRepository) ProfileService {
	return &profileService{repo: repo}
}

func (s *profileService) GetMyProfile(ctx context.Context, myID string) (*model.UserProfile, error) {
	return s.repo.GetProfileByID(ctx, myID)
}

func (s *profileService) UpdateMyProfile(ctx context.Context, myID string, req model.UpdateProfileRequest) error {
	return s.repo.UpdateProfile(ctx, myID, req)
}

func (s *profileService) GetUserProfile(ctx context.Context, myID, targetID string) (map[string]interface{}, error) {
	status, err := s.repo.GetFriendshipStatus(ctx, myID, targetID)
	if err != nil {
		return nil, err
	}
	isFriend := (status == "accepted")

	p, err := s.repo.GetProfileByID(ctx, targetID)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"id":          p.ID,
		"public_info": p.PublicInfo,
	}

	if isFriend {
		payload["friends_info"] = p.FriendsInfo
	}

	return payload, nil
}
