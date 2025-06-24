package model

type Order struct {
	ID       string `json:"id"`
	ItemName string `json:"watch"`
	Success  bool   `json:"success"`
	Reason   string `json:"reason,omitempty"`
}
