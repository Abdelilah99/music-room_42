package model

import "encoding/json"


type UserProfile struct {
	ID               string          `json:"id"`
	Email            string          `json:"email"`
	PublicInfo       json.RawMessage `json:"public_info"`
	FriendsInfo      json.RawMessage `json:"friends_info,omitempty"`
	PrivateInfo      json.RawMessage `json:"private_info,omitempty"`
	MusicPreferences json.RawMessage `json:"music_preferences,omitempty"`
}


type UpdateProfileRequest struct {
	PublicInfo       *json.RawMessage `json:"public_info"`
	FriendsInfo      *json.RawMessage `json:"friends_info"`
	PrivateInfo      *json.RawMessage `json:"private_info"`
	MusicPreferences *json.RawMessage `json:"music_preferences"`
}
