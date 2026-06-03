package handler

import (
	"errors"
	"net/http"

	"music-room/internal/middleware"
	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FriendHandler struct {
	svc service.FriendService
}

func NewFriendHandler(svc service.FriendService) *FriendHandler {
	return &FriendHandler{svc: svc}
}

func (h *FriendHandler) callerID(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return uuid.Nil, false
	}
	return id, true
}

func (h *FriendHandler) SendRequest(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	var body struct {
		AddresseeID string `json:"addressee_id" binding:"required,uuid"`
	}
	if !middleware.BindAndValidate(c, &body) {
		return
	}

	addresseeID, _ := uuid.Parse(body.AddresseeID)

	f, err := h.svc.SendRequest(c.Request.Context(), callerID, addresseeID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCannotFriendSelf):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrFriendshipExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"friendship_id": f.ID,
		"status":        f.Status,
	})
}

func (h *FriendHandler) AcceptRequest(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	friendshipID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship id"})
		return
	}

	if err := h.svc.AcceptRequest(c.Request.Context(), friendshipID, callerID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "friend request accepted"})
}

func (h *FriendHandler) RejectRequest(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	friendshipID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship id"})
		return
	}

	if err := h.svc.RejectRequest(c.Request.Context(), friendshipID, callerID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "friend request rejected"})
}

func (h *FriendHandler) Unfriend(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	friendshipID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship id"})
		return
	}

	if err := h.svc.Unfriend(c.Request.Context(), friendshipID, callerID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "unfriended successfully"})
}

func (h *FriendHandler) ListFriends(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	friends, err := h.svc.ListFriends(c.Request.Context(), callerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	if friends == nil {
		friends = []model.FriendEntry{}
	}

	c.JSON(http.StatusOK, gin.H{"friends": friends})
}

func (h *FriendHandler) ListRequests(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	requests, err := h.svc.ListIncomingRequests(c.Request.Context(), callerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	if requests == nil {
		requests = []model.FriendEntry{}
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (h *FriendHandler) ListOutgoing(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	requests, err := h.svc.ListOutgoingRequests(c.Request.Context(), callerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}

	if requests == nil {
		requests = []model.FriendEntry{}
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (h *FriendHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrFriendshipNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotAddresseeOp),
		errors.Is(err, service.ErrNotParticipantOp):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrRequestNotPending),
		errors.Is(err, service.ErrRequestNotAccepted):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
	}
}
