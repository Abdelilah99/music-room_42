package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// QueueBroadcaster pushes a serialized message to every client watching an
// event's hub. Satisfied by hub.HubManager.
type QueueBroadcaster interface {
	Broadcast(entityID string, msg []byte)
}

// queueUpdate is the WebSocket frame sent after a vote. Its data field carries
// the same track shape as GET /events/:id/queue so clients can replace state.
type queueUpdate struct {
	Type string        `json:"type"`
	Data []model.Track `json:"data"`
}

type TrackHandler struct {
	svc         service.TrackService
	broadcaster QueueBroadcaster
}

func NewTrackHandler(svc service.TrackService, broadcaster QueueBroadcaster) *TrackHandler {
	return &TrackHandler{svc: svc, broadcaster: broadcaster}
}

func (h *TrackHandler) Suggest(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	var req model.SuggestTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external_id, title and artist are required"})
		return
	}

	track, err := h.svc.Suggest(c.Request.Context(), eventID, callerID, req)
	if err != nil {
		h.handleTrackError(c, err)
		return
	}

	c.JSON(http.StatusCreated, track)
}

func (h *TrackHandler) GetQueue(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	tracks, err := h.svc.GetQueue(c.Request.Context(), eventID, callerID)
	if err != nil {
		h.handleTrackError(c, err)
		return
	}

	if tracks == nil {
		tracks = []model.Track{}
	}

	c.JSON(http.StatusOK, gin.H{"tracks": tracks})
}

func (h *TrackHandler) Vote(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	trackID, err := uuid.Parse(c.Param("trackId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "track not found"})
		return
	}

	var req model.VoteRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
	}

	if err := h.svc.Vote(c.Request.Context(), eventID, trackID, callerID, req); err != nil {
		h.handleTrackError(c, err)
		return
	}

	h.broadcastQueue(c, eventID, callerID)

	c.JSON(http.StatusOK, gin.H{"message": "vote cast"})
}

// broadcastQueue pushes the updated queue to every client watching the event.
// Best-effort: the vote has already been recorded, so any failure here is
// swallowed rather than turned into an error response.
func (h *TrackHandler) broadcastQueue(c *gin.Context, eventID, callerID uuid.UUID) {
	if h.broadcaster == nil {
		return
	}

	tracks, err := h.svc.GetQueue(c.Request.Context(), eventID, callerID)
	if err != nil {
		return
	}
	if tracks == nil {
		tracks = []model.Track{}
	}

	payload, err := json.Marshal(queueUpdate{Type: "queue_update", Data: tracks})
	if err != nil {
		return
	}

	h.broadcaster.Broadcast(eventID.String(), payload)
}

func (h *TrackHandler) DeleteTrack(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	trackID, err := uuid.Parse(c.Param("trackId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "track not found"})
		return
	}

	if err := h.svc.DeleteTrack(c.Request.Context(), eventID, trackID, callerID); err != nil {
		h.handleTrackError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "track removed"})
}

func (h *TrackHandler) handleTrackError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrTrackNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrTrackAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrAlreadyVoted):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotInvited):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrOutOfRange):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrVotingClosed):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrMissingCoords):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
	}
}

func extractCallerID(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}
	return id, true
}
