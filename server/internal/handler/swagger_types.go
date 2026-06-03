package handler

// Shared response types used only for Swagger annotation schemas.
// The actual handlers return gin.H; these structs mirror their shape.

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type FriendshipCreatedResponse struct {
	FriendshipID string `json:"friendship_id"`
	Status       string `json:"status"`
}

//lint:ignore U1000 used in swagger annotations
type sendFriendRequestBody struct {
	AddresseeID string `json:"addressee_id" binding:"required,uuid"`
}
