package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"music-room/internal/model"

	"github.com/gin-gonic/gin"
)

// MockMusicService is a mock implementation of service.MusicService
type MockMusicService struct {
	SearchTracksFunc func(ctx context.Context, query string) ([]model.TrackDTO, error)
}

func (m *MockMusicService) SearchTracks(ctx context.Context, query string) ([]model.TrackDTO, error) {
	return m.SearchTracksFunc(ctx, query)
}

func setupMusicRouter(mockSvc *MockMusicService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewMusicHandler(mockSvc)
	r.GET("/api/v1/music/search", handler.Search)
	return r
}

func TestSearchMusic(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		mockResponse   []model.TrackDTO
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Empty query returns 400",
			query:          "",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"code":"INVALID_REQUEST","error":"missing or empty 'q' parameter"}`,
		},
		{
			name:           "Whitespace query returns 400",
			query:          "   ",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"code":"INVALID_REQUEST","error":"missing or empty 'q' parameter"}`,
		},
		{
			name:  "Successful search returns 200 and track array",
			query: "radiohead",
			mockResponse: []model.TrackDTO{
				{
					ExternalID: "138547415",
					Title:      "Creep",
					Artist:     "Radiohead",
					Album:      "Pablo Honey",
					PreviewURL: "http://example.com/preview.mp3",
					CoverURL:   "http://example.com/cover.jpg",
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"external_id":"138547415","title":"Creep","artist":"Radiohead","album":"Pablo Honey","preview_url":"http://example.com/preview.mp3","cover_url":"http://example.com/cover.jpg"}]`,
		},
		{
			name:           "Upstream service error returns 502",
			query:          "unknown_error",
			mockResponse:   nil,
			mockError:      errors.New("deezer API returned status 500"),
			expectedStatus: http.StatusBadGateway,
			expectedBody:   `{"code":"BAD_GATEWAY","error":"failed to proxy search request"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockMusicService{
				SearchTracksFunc: func(ctx context.Context, query string) ([]model.TrackDTO, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			router := setupMusicRouter(mockSvc)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/music/search?q="+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			// Validate JSON structure directly if not purely matching body string
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				// To handle JSON formatting differences, unmarshal vs marshal comparison is better
				// But exact matches work too here, so string matching should suffice
				var expected, actual interface{}
				json.Unmarshal([]byte(tt.expectedBody), &expected)
				json.Unmarshal(w.Body.Bytes(), &actual)
				
				expBytes, _ := json.Marshal(expected)
				actBytes, _ := json.Marshal(actual)

				if string(expBytes) != string(actBytes) {
					t.Errorf("Expected body %s, got %s", string(expBytes), string(actBytes))
				}
			}
		})
	}
}
