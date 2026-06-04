package handler

import (
	"net/http"
	"strings"

	"music-room/internal/service"

	"github.com/gin-gonic/gin"
)

type MusicHandler struct {
	musicSvc service.MusicService
}

func NewMusicHandler(musicSvc service.MusicService) *MusicHandler {
	return &MusicHandler{
		musicSvc: musicSvc,
	}
}

// Search godoc
// @Summary      Search tracks via Deezer
// @Tags         music
// @Produce      json
// @Security     BearerAuth
// @Param        q query string true "Search query"
// @Success      200 {array}  music-room_internal_model.TrackDTO
// @Failure      400 {object} ErrorResponse "Missing query"
// @Failure      401 {object} ErrorResponse
// @Failure      502 {object} ErrorResponse "Deezer unreachable"
// @Router       /music/search [get]
func (h *MusicHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing or empty 'q' parameter",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	results, err := h.musicSvc.SearchTracks(c.Request.Context(), query)
	if err != nil {
		// As required by acceptance criteria: return 502 if Deezer is unreachable or returns an error
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "failed to proxy search request",
			"code":  "BAD_GATEWAY",
		})
		return
	}

	c.JSON(http.StatusOK, results)
}
