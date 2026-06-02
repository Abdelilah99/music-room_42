package handler

import (
	"net/http"

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
