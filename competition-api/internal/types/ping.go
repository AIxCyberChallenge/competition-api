package types

type PingResponse struct {
	Status string `json:"status" validate:"required"`
}
