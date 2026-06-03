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

// SendRequest godoc
// @Summary      Send a friend request
// @Tags         friends
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body sendFriendRequestBody true "Target user UUID"
// @Success      201 {object} FriendshipCreatedResponse
// @Failure      400 {object} ErrorResponse "Cannot friend yourself"
// @Failure      401 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse "Friendship already exists"
// @Failure      500 {object} ErrorResponse
// @Router       /friends/request [post]
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

// AcceptRequest godoc
// @Summary      Accept a friend request
// @Tags         friends
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Friendship UUID"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse "Invalid UUID"
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse "Not the addressee"
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse "Request not pending"
// @Router       /friends/accept/{id} [post]
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

// RejectRequest godoc
// @Summary      Reject or cancel a friend request
// @Tags         friends
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Friendship UUID"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /friends/reject/{id} [delete]
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

// Unfriend godoc
// @Summary      Remove a friend
// @Tags         friends
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Friendship UUID"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /friends/{id} [delete]
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

// ListFriends godoc
// @Summary      List accepted friends
// @Tags         friends
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string][]model.FriendEntry
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /friends [get]
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

// ListRequests godoc
// @Summary      List incoming friend requests
// @Tags         friends
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string][]model.FriendEntry
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /friends/requests [get]
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

// ListOutgoing godoc
// @Summary      List outgoing (sent) friend requests
// @Tags         friends
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string][]model.FriendEntry
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /friends/outgoing [get]
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
