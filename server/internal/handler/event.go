package handler

import (
	"errors"
	"net/http"
	"strconv"

	"music-room/internal/hub"
	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EventHandler struct {
	svc        service.EventService
	hubManager *hub.HubManager
}

func NewEventHandler(svc service.EventService, hubManager *hub.HubManager) *EventHandler {
	return &EventHandler{svc: svc, hubManager: hubManager}
}

func (h *EventHandler) callerID(c *gin.Context) (uuid.UUID, bool) {
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

func (h *EventHandler) Create(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	var req model.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	event, err := h.svc.Create(c.Request.Context(), callerID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, event)
}

func (h *EventHandler) List(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	f := model.EventListFilter{Q: c.Query("q")}

	if lat, lng, radius, ok := parseLocationParams(c); ok {
		f.Lat = &lat
		f.Lng = &lng
		f.Radius = &radius
	}

	events, err := h.svc.List(c.Request.Context(), callerID, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
		return
	}

	if events == nil {
		events = []model.Event{}
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (h *EventHandler) Get(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	event, err := h.svc.Get(c.Request.Context(), eventID, callerID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, event)
}

func (h *EventHandler) Update(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	var req model.UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	event, err := h.svc.Update(c.Request.Context(), eventID, callerID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, event)
}

func (h *EventHandler) Delete(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), eventID, callerID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "event deleted"})
}

func (h *EventHandler) Invite(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	var body struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	if _, err := uuid.Parse(body.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be a valid UUID"})
		return
	}

	targetUserID, _ := uuid.Parse(body.UserID)

	if err := h.svc.Invite(c.Request.Context(), eventID, callerID, targetUserID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user invited"})
}

// ServeWS upgrades to WebSocket for the event's real-time queue hub. Only users
// who can access the event (owner, invited, or public license) are allowed in.
func (h *EventHandler) ServeWS(c *gin.Context) {
	callerID, ok := h.callerID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	if _, err := h.svc.Get(c.Request.Context(), eventID, callerID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	hub.ServeWS(h.hubManager, eventID.String(), c)
}

func (h *EventHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrAlreadyInvited):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidLicenseConfig):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
	}
}

// parseLocationParams parses ?lat=&lng=&radius= query params.
// Returns ok=false if any of the three are missing or invalid.
func parseLocationParams(c *gin.Context) (lat, lng, radius float64, ok bool) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	radiusStr := c.Query("radius")

	if latStr == "" || lngStr == "" || radiusStr == "" {
		return 0, 0, 0, false
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	lng, err = strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	radius, err = strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	return lat, lng, radius, true
}
