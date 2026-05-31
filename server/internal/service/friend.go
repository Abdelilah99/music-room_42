package service

import (
	"context"
	"errors"

	"music-room/internal/model"
	"music-room/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrCannotFriendSelf    = errors.New("cannot send a friend request to yourself")
	ErrFriendshipExists    = errors.New("a pending or accepted friendship already exists")
	ErrFriendshipNotFound  = errors.New("friendship not found")
	ErrNotAddresseeOp      = errors.New("only the request recipient can perform this action")
	ErrNotParticipantOp    = errors.New("you are not part of this friendship")
	ErrRequestNotPending   = errors.New("friendship request is not in pending state")
	ErrRequestNotAccepted  = errors.New("friendship is not in accepted state")
)

type FriendService interface {
	SendRequest(ctx context.Context, requesterID, addresseeID uuid.UUID) (*model.Friendship, error)
	AcceptRequest(ctx context.Context, friendshipID, callerID uuid.UUID) error
	RejectRequest(ctx context.Context, friendshipID, callerID uuid.UUID) error
	Unfriend(ctx context.Context, friendshipID, callerID uuid.UUID) error
	ListFriends(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error)
	ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error)
}

type friendService struct {
	repo repository.FriendRepository
}

func NewFriendService(repo repository.FriendRepository) FriendService {
	return &friendService{repo: repo}
}

func (s *friendService) SendRequest(ctx context.Context, requesterID, addresseeID uuid.UUID) (*model.Friendship, error) {
	if requesterID == addresseeID {
		return nil, ErrCannotFriendSelf
	}

	existing, err := s.repo.GetExisting(ctx, requesterID, addresseeID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrFriendshipExists
	}

	return s.repo.Create(ctx, requesterID, addresseeID)
}

func (s *friendService) AcceptRequest(ctx context.Context, friendshipID, callerID uuid.UUID) error {
	f, err := s.repo.GetByID(ctx, friendshipID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrFriendshipNotFound
		}
		return err
	}

	if f.AddresseeID != callerID {
		return ErrNotAddresseeOp
	}
	if f.Status != "pending" {
		return ErrRequestNotPending
	}

	return s.repo.Accept(ctx, friendshipID)
}

func (s *friendService) RejectRequest(ctx context.Context, friendshipID, callerID uuid.UUID) error {
	f, err := s.repo.GetByID(ctx, friendshipID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrFriendshipNotFound
		}
		return err
	}

	if f.AddresseeID != callerID {
		return ErrNotAddresseeOp
	}
	if f.Status != "pending" {
		return ErrRequestNotPending
	}

	return s.repo.Delete(ctx, friendshipID)
}

func (s *friendService) Unfriend(ctx context.Context, friendshipID, callerID uuid.UUID) error {
	f, err := s.repo.GetByID(ctx, friendshipID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrFriendshipNotFound
		}
		return err
	}

	if f.RequesterID != callerID && f.AddresseeID != callerID {
		return ErrNotParticipantOp
	}
	if f.Status != "accepted" {
		return ErrRequestNotAccepted
	}

	return s.repo.Delete(ctx, friendshipID)
}

func (s *friendService) ListFriends(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error) {
	return s.repo.ListFriends(ctx, userID)
}

func (s *friendService) ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]model.FriendEntry, error) {
	return s.repo.ListIncomingRequests(ctx, userID)
}
