package model

type CommandRequest struct {
	Action string `json:"action" binding:"required"`
	Value  *int   `json:"value"`
}

type CommandBroadcast struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Value  *int   `json:"value"`
	SentBy string `json:"sent_by"`
}
