package model

type CommandRequest struct {
	Action string  `json:"action" binding:"required"`
	Value  *int    `json:"value"`
	Track  *string `json:"track"` // now-playing label, sent by the owner on a "track" action
}

type CommandBroadcast struct {
	Type   string  `json:"type"`
	Action string  `json:"action"`
	Value  *int    `json:"value"`
	Track  *string `json:"track,omitempty"`
	SentBy string  `json:"sent_by"`
}
