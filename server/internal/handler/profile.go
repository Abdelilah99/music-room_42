package handler

import (
	"context"
	"net/http"

	"music-room/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProfileHandler struct {
	pool *pgxpool.Pool
}

func NewProfileHandler(pool *pgxpool.Pool) *ProfileHandler {
	return &ProfileHandler{pool: pool}
}

// GET /api/v1/users/me
func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	myID := c.MustGet("authenticated_user_id").(string)

	var p model.UserProfile
	query := `SELECT id, email, public_info, friends_info, private_info, music_preferences 
	          FROM users WHERE id = $1`

	err := h.pool.QueryRow(context.Background(), query, myID).Scan(
		&p.ID, &p.Email, &p.PublicInfo, &p.FriendsInfo, &p.PrivateInfo, &p.MusicPreferences,
	)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User account not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, p)
}

// PATCH /api/v1/users/me
func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	myID := c.MustGet("authenticated_user_id").(string)

	var req model.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload representation"})
		return
	}

	ctx := context.Background()

	// Direct execution blocks ensuring zero updates to non-provided keys
	if req.PublicInfo != nil {
		if _, err := h.pool.Exec(ctx, "UPDATE users SET public_info = $1 WHERE id = $2", *req.PublicInfo, myID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.FriendsInfo != nil {
		if _, err := h.pool.Exec(ctx, "UPDATE users SET friends_info = $1 WHERE id = $2", *req.FriendsInfo, myID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.PrivateInfo != nil {
		if _, err := h.pool.Exec(ctx, "UPDATE users SET private_info = $1 WHERE id = $2", *req.PrivateInfo, myID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.MusicPreferences != nil {
		if _, err := h.pool.Exec(ctx, "UPDATE users SET music_preferences = $1 WHERE id = $2", *req.MusicPreferences, myID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "Profile structural blocks saved successfully"})
}

// GET /api/v1/users/:id
func (h *ProfileHandler) GetUserProfile(c *gin.Context) {
	myID := c.MustGet("authenticated_user_id").(string)
	targetID := c.Param("id")

	if myID == targetID {
		h.GetMyProfile(c)
		return
	}

	// 1. Relational state query utilizing the friendships schema
	var status string
	relQuery := `SELECT status FROM friendships 
	             WHERE (requester_id = $1 AND addressee_id = $2) 
	                OR (requester_id = $2 AND addressee_id = $1)`
	
	err := h.pool.QueryRow(context.Background(), relQuery, myID, targetID).Scan(&status)
	isFriend := (err == nil && status == "accepted")

	// 2. Resource profile lookup
	var p model.UserProfile
	query := `SELECT id, public_info, friends_info FROM users WHERE id = $1`
	err = h.pool.QueryRow(context.Background(), query, targetID).Scan(&p.ID, &p.PublicInfo, &p.FriendsInfo)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target user not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3. Dynamic masking validation enforcement
	payload := gin.H{
		"id":          p.ID,
		"public_info": p.PublicInfo,
	}

	if isFriend {
		payload["friends_info"] = p.FriendsInfo
	}

	c.JSON(http.StatusOK, payload)
}
