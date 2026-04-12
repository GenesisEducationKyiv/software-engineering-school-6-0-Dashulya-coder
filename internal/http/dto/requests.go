package dto

type SubscribeRequest struct {
	Email string `json:"email"`
	Repo  string `json:"repo"`
}
