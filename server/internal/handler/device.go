package handler

import (
	"errors"
	"net/http"

	"music-room/internal/model"
	"music-room/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DeviceHandler struct {
	svc service.DeviceService
}

func NewDeviceHandler(svc service.DeviceService) *DeviceHandler {
	return &DeviceHandler{svc: svc}
}

func (h *DeviceHandler) Register(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	var req model.CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, platform and model are required"})
		return
	}

	device, err := h.svc.Register(c.Request.Context(), callerID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, device)
}

func (h *DeviceHandler) List(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	devices, err := h.svc.List(c.Request.Context(), callerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
		return
	}

	if devices == nil {
		devices = []model.DeviceWithDelegate{}
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func (h *DeviceHandler) Get(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	device, err := h.svc.Get(c.Request.Context(), deviceID, callerID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, device)
}

func (h *DeviceHandler) Delete(c *gin.Context) {
	callerID, ok := extractCallerID(c)
	if !ok {
		return
	}

	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), deviceID, callerID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device deleted"})
}

func (h *DeviceHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDeviceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrDeviceAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
	}
}
