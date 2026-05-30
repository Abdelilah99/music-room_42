package auth

import (
	"context"
	"music-room/internal/model"
)

func (h *Handler) TestGenerateTokenPair(ctx context.Context, user *model.User) (string, string, error) {
	return h.generateTokenPair(ctx, user)
}
