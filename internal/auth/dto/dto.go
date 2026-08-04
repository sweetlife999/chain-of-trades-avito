package dto

type LoginRequest struct {
	Nickname string `json:"nickname"`
	Password string `json:"password"`
}
