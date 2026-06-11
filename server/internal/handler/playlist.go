package handler

import (
	"errors"
	"net/http"

	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PlaylistHandler struct {
	svc service.PlaylistService
}

func NewPlaylistHandler(svc service.PlaylistService) *PlaylistHandler {
	return &PlaylistHandler{svc: svc}
}

func (h *PlaylistHandler) Create(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	var req model.CreatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	playlist, err := h.svc.Create(c.Request.Context(), callerID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, playlist)
}

func (h *PlaylistHandler) List(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	f := model.PlaylistListFilter{Q: c.Query("q")}

	playlists, err := h.svc.List(c.Request.Context(), callerID, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
		return
	}

	if playlists == nil {
		playlists = []model.Playlist{}
	}

	c.JSON(http.StatusOK, gin.H{"playlists": playlists})
}

func (h *PlaylistHandler) Get(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	playlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	playlist, err := h.svc.Get(c.Request.Context(), playlistID, callerID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, playlist)
}

func (h *PlaylistHandler) Update(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	playlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	var req model.UpdatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	playlist, err := h.svc.Update(c.Request.Context(), playlistID, callerID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, playlist)
}

func (h *PlaylistHandler) Delete(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	playlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), playlistID, callerID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "playlist deleted"})
}

func (h *PlaylistHandler) Invite(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	playlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	var body struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	targetUserID, err := uuid.Parse(body.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be a valid UUID"})
		return
	}

	if err := h.svc.Invite(c.Request.Context(), playlistID, callerID, targetUserID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user invited"})
}

func (h *PlaylistHandler) AddTrack(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	playlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	var req model.AddTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "external_id, title, and artist are required"})
		return
	}

	track, err := h.svc.AddTrack(c.Request.Context(), playlistID, callerID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, track)
}

func (h *PlaylistHandler) RemoveTrack(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	playlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	trackID, err := uuid.Parse(c.Param("trackId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "track not found"})
		return
	}

	if err := h.svc.RemoveTrack(c.Request.Context(), playlistID, callerID, trackID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "track removed"})
}

func (h *PlaylistHandler) MoveTrack(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	playlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	trackID, err := uuid.Parse(c.Param("trackId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "track not found"})
		return
	}

	var req model.MoveTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "position must be a positive integer"})
		return
	}

	if err := h.svc.MoveTrack(c.Request.Context(), playlistID, callerID, trackID, req.Position); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "track moved"})
}

func (h *PlaylistHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPlaylistNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrPlaylistTrackNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrEditNotAllowed):
		c.JSON(http.StatusForbidden, gin.H{"code": "EDIT_NOT_ALLOWED", "error": err.Error()})
	case errors.Is(err, service.ErrDuplicateTrack):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidPosition):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
	}
}
