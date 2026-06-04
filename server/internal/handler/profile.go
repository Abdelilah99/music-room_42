package handler

import (
	"net/http"
	"strings"

	"music-room/internal/middleware"
	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type ProfileHandler struct {
	svc service.ProfileService
}

func NewProfileHandler(svc service.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

// GetMyProfile godoc
// @Summary      Get the authenticated user's profile
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} model.UserProfile
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /users/me [get]
func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	ctx := c.Request.Context() // Support cancellation via client context
	myID := c.MustGet("user_id").(string)

	p, err := h.svc.GetMyProfile(ctx, myID)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User profile not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	c.JSON(http.StatusOK, p)
}

// UpdateMyProfile godoc
// @Summary      Update the authenticated user's profile
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body model.UpdateProfileRequest true "Fields to update (all optional)"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /users/me [patch]
func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	ctx := c.Request.Context()
	myID := c.MustGet("user_id").(string)

	var req model.UpdateProfileRequest
	if !middleware.BindAndValidate(c, &req) {
		return
	}

	if err := h.svc.UpdateMyProfile(ctx, myID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Profile updated successfully"})
}

// SearchUsers godoc
// @Summary      Search users by name or email
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        q query string true "Search query"
// @Success      200 {object} map[string][]model.UserSearchResult
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /users/search [get]
func (h *ProfileHandler) SearchUsers(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, gin.H{"users": []any{}})
		return
	}

	results, err := h.svc.SearchUsers(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	if results == nil {
		results = []model.UserSearchResult{}
	}

	c.JSON(http.StatusOK, gin.H{"users": results})
}

// GetUserProfile godoc
// @Summary      Get another user's profile
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Target user UUID"
// @Success      200 {object} model.UserProfile
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /users/{id} [get]
func (h *ProfileHandler) GetUserProfile(c *gin.Context) {
	ctx := c.Request.Context()
	myID := c.MustGet("user_id").(string)
	targetID := c.Param("id")

	if myID == targetID {
		h.GetMyProfile(c)
		return
	}

	payload, err := h.svc.GetUserProfile(ctx, myID, targetID)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target user profile not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	c.JSON(http.StatusOK, payload)
}
