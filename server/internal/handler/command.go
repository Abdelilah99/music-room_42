package handler

import (
	"encoding/json"
	"net/http"

	"music-room/internal/hub"
	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var validActions = map[string]bool{
	"play":   true,
	"pause":  true,
	"next":   true,
	"track":  true, // metadata only: carries the owner's now-playing label
	"volume": true,
}

type CommandHandler struct {
	deviceSvc  service.DeviceService
	hubManager *hub.HubManager
}

func NewCommandHandler(deviceSvc service.DeviceService, hubManager *hub.HubManager) *CommandHandler {
	return &CommandHandler{deviceSvc: deviceSvc, hubManager: hubManager}
}

func (h *CommandHandler) Send(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	var req model.CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action is required"})
		return
	}

	if !validActions[req.Action] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be one of: play, pause, next, volume"})
		return
	}

	if req.Action == "volume" {
		if req.Value == nil || *req.Value < 0 || *req.Value > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "volume requires a value between 0 and 100"})
			return
		}
	}

	broadcast := model.CommandBroadcast{
		Type:   "command",
		Action: req.Action,
		Value:  req.Value,
		Track:  req.Track,
		SentBy: callerID.String(),
	}

	payload, err := json.Marshal(broadcast)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
		return
	}

	h.hubManager.Broadcast(deviceID.String(), payload)
	c.JSON(http.StatusOK, gin.H{"message": "command sent"})
}

func (h *CommandHandler) ServeDeviceWS(c *gin.Context) {
	if _, ok := extractCallerID(c); !ok {
		return
	}

	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	// Authorization (owner or active delegate) is enforced by the
	// RequireDelegateOrOwner middleware on this route, so both can subscribe
	// and stay in sync with each other's commands.
	hub.ServeWS(h.hubManager, deviceID.String(), c)
}
