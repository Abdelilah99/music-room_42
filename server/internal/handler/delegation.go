package handler

import (
	"errors"
	"net/http"

	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DelegationHandler struct {
	svc service.DelegationService
}

func NewDelegationHandler(svc service.DelegationService) *DelegationHandler {
	return &DelegationHandler{svc: svc}
}

func (h *DelegationHandler) Grant(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	var req model.DelegateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "friend_user_id is required"})
		return
	}

	friendID, err := uuid.Parse(req.FriendUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "friend_user_id must be a valid UUID"})
		return
	}

	if err := h.svc.Grant(c.Request.Context(), deviceID, callerID, friendID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "delegation granted"})
}

func (h *DelegationHandler) Revoke(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	if err := h.svc.Revoke(c.Request.Context(), deviceID, callerID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "delegation revoked"})
}

func (h *DelegationHandler) ListDelegated(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	devices, err := h.svc.ListDelegated(c.Request.Context(), callerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
		return
	}

	if devices == nil {
		devices = []model.DelegatedDevice{}
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// RequireDelegateOrOwner is a Gin middleware that enforces the caller is either
// the device owner or the user holding an active delegation for the device.
// Apply this to any route that takes :id as a device ID and requires delegation auth.
func (h *DelegationHandler) RequireDelegateOrOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID, ok := extractCallerID(c)
		if !ok {
			c.Abort()
			return
		}

		deviceID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			c.Abort()
			return
		}

		authorized, err := h.svc.IsAuthorized(c.Request.Context(), deviceID, callerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
			c.Abort()
			return
		}
		if !authorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "NOT_AUTHORIZED"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (h *DelegationHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDeviceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotFriends):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotAuthorized):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
	}
}
