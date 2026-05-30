package handler

import (
	"net/http"

	"music-room/internal/model"
	"music-room/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type ProfileHandler struct {
	repo repository.ProfileRepository
}

func NewProfileHandler(repo repository.ProfileRepository) *ProfileHandler {
	return &ProfileHandler{repo: repo}
}

// GET /api/v1/users/me
func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	// Switch out context.Background() for c.Request.Context() to support client cancellation
	ctx := c.Request.Context()
	myID := c.MustGet("authenticated_user_id").(string)

	p, err := h.repo.GetProfileByID(ctx, myID)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User profile not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	c.JSON(http.StatusOK, p)
}

// PATCH /api/v1/users/me
func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	ctx := c.Request.Context()
	myID := c.MustGet("authenticated_user_id").(string)

	var req model.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload representation"})
		return
	}

	if err := h.repo.UpdateProfile(ctx, myID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Profile updated successfully"})
}

// GET /api/v1/users/:id
func (h *ProfileHandler) GetUserProfile(c *gin.Context) {
	ctx := c.Request.Context()
	myID := c.MustGet("authenticated_user_id").(string)
	targetID := c.Param("id")

	if myID == targetID {
		h.GetMyProfile(c)
		return
	}

	status, err := h.repo.GetFriendshipStatus(ctx, myID, targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}
	isFriend := (status == "accepted")

	p, err := h.repo.GetProfileByID(ctx, targetID)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target user profile not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	payload := gin.H{
		"id":          p.ID,
		"public_info": p.PublicInfo,
	}

	if isFriend {
		payload["friends_info"] = p.FriendsInfo
	}

	c.JSON(http.StatusOK, payload)
}
