package handler

import (
	"errors"
	"net/http"

	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TrackHandler struct {
	svc service.TrackService
}

func NewTrackHandler(svc service.TrackService) *TrackHandler {
	return &TrackHandler{svc: svc}
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

	c.JSON(http.StatusOK, gin.H{"message": "vote cast"})
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
