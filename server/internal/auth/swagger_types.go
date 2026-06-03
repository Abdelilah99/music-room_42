package auth

// Shared response types used only for Swagger annotation schemas.

//lint:ignore U1000 used in swagger annotations
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

//lint:ignore U1000 used in swagger annotations
type messageResponse struct {
	Message string `json:"message"`
}

//lint:ignore U1000 used in swagger annotations
type errorResponse struct {
	Error string `json:"error"`
}
