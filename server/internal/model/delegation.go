package model

type DelegateRequest struct {
	FriendUserID string `json:"friend_user_id" binding:"required"`
}

type DelegatedDevice struct {
	Device
	Owner ActiveDelegate `json:"owner"`
}
